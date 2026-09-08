#[cfg(test)]
mod integration_test {
    use crate::contract::{execute, instantiate, query};
    use crate::msgs::{ExecuteMsg, InstantiateMsg, QueryMsg};
    use crate::state::{EpochInfo, Funding};

    use cosmwasm_std::{from_json, Addr, Coin};
    use cw_multi_test::{App, BankSudo, ContractWrapper, Executor};
    use easy_addr::addr;
    use nibiru_std::errors::TestResult;

    const ADDR_ROOT: &str = addr!("root");

    #[derive(Debug, Clone)]
    pub struct TestContracts {
        pub contract_incentives_addr: Addr,
        pub contract_lockup_addr: Addr,
    }
    pub struct TestDeps {
        pub app: App,
        pub contracts: TestContracts,
    }

    fn create_program(
        app: &mut App,
        contracts: TestContracts,
        denom: String,
        epochs: u64,
        epoch_block_duration: u64,
        min_lockup_blocks: u64,
    ) {
        app.execute_contract(
            Addr::unchecked(ADDR_ROOT),
            contracts.contract_incentives_addr,
            &ExecuteMsg::CreateProgram {
                denom,
                epochs,
                epoch_block_duration,
                min_lockup_blocks,
            },
            &[],
        )
        .unwrap();
    }

    fn fund_program(
        app: &mut App,
        contracts: TestContracts,
        _program_id: u64,
        coins: &[Coin],
    ) -> Vec<Coin> {
        app.execute_contract(
            Addr::unchecked(ADDR_ROOT),
            contracts.contract_incentives_addr,
            &ExecuteMsg::FundProgram { id: 1 },
            coins,
        )
        .unwrap();

        app.wrap().query_all_balances(ADDR_ROOT).unwrap()
    }

    fn mint(app: &mut App, to: &Addr, coins: &[Coin]) {
        app.sudo(
            BankSudo::Mint {
                to_address: to.to_string(),
                amount: coins.to_vec(),
            }
            .into(),
        )
        .unwrap();
    }

    fn mint_and_lock(
        app: &mut App,
        contracts: TestContracts,
        user: &Addr,
        coins: &[Coin],
        blocks: u64,
    ) {
        mint(app, user, coins);
        // we make alice lock some atoms
        app.execute_contract(
            user.clone(),
            contracts.contract_lockup_addr,
            &lockup::msgs::ExecuteMsg::Lock { blocks },
            coins,
        )
        .unwrap();
    }

    fn withdraw_rewards(
        app: &mut App,
        contracts: TestContracts,
        user: &Addr,
        _program_id: u64,
    ) -> Vec<Coin> {
        app.execute_contract(
            user.clone(),
            contracts.contract_incentives_addr,
            &crate::msgs::ExecuteMsg::WithdrawRewards { id: 1 },
            &[],
        )
        .unwrap();

        app.wrap().query_all_balances(user).unwrap()
    }

    fn process_epoch(
        app: &mut App,
        contracts: TestContracts,
        _program_id: u64,
    ) -> EpochInfo {
        from_json::<EpochInfo>(
            &app.execute_contract(
                Addr::unchecked(ADDR_ROOT),
                contracts.contract_incentives_addr,
                &ExecuteMsg::ProcessEpoch { id: 1 },
                &[],
            )
            .unwrap()
            .data
            .unwrap(),
        )
        .unwrap()
    }

    fn app() -> anyhow::Result<TestDeps> {
        let mut app = App::default();
        // note don't break the order otherwise contracts will have different addresses
        // which renders the const in the module useless TODO: maybe do better.

        let lockup_contract = Box::new(ContractWrapper::new(
            lockup::contract::execute,
            lockup::contract::instantiate,
            lockup::contract::query,
        ));
        let code = app.store_code(lockup_contract);
        let contract_lockup_addr = app.instantiate_contract(
            code,
            Addr::unchecked(ADDR_ROOT),
            &lockup::msgs::InstantiateMsg {},
            &[],
            "lockup",
            None,
        )?;

        let incentives_contract =
            Box::new(ContractWrapper::new(execute, instantiate, query));
        let code = app.store_code(incentives_contract);
        let contract_incentives_addr = app.instantiate_contract(
            code,
            Addr::unchecked(ADDR_ROOT),
            &InstantiateMsg {
                lockup_contract_address: contract_lockup_addr.clone(),
            },
            &[],
            "incentives",
            None,
        )?;

        Ok(TestDeps {
            app,
            contracts: TestContracts {
                contract_incentives_addr,
                contract_lockup_addr,
            },
        })
    }

    #[test]
    fn flow() -> TestResult {
        let test_deps = app()?;
        let mut app = test_deps.app;
        let contracts = test_deps.contracts;

        let alice = Addr::unchecked(addr!("alice"));
        let bob = Addr::unchecked(addr!("bob"));

        // mint coins
        mint(
            &mut app,
            &Addr::unchecked(ADDR_ROOT),
            &[
                Coin::new(1_000_000u128, "ATOM"),
                Coin::new(1_000_000u128, "OSMO"),
                Coin::new(1_000_000u128, "LUNA"),
            ],
        );

        // we make alice lock some lp coins
        let blocks = 100;
        mint_and_lock(
            &mut app,
            contracts.clone(),
            &alice,
            &[Coin::new(100u128, "NIBI_LP")],
            blocks,
        );

        // now we create a new incentives program
        create_program(
            &mut app,
            contracts.clone(),
            "NIBI_LP".to_string(),
            5,
            5,
            50,
        );

        // now we fund the incentives program
        let balance = fund_program(
            &mut app,
            contracts.clone(),
            1,
            &[Coin::new(1_000u128, "ATOM")],
        );

        println!("{:?}", balance,);

        let funding: Vec<Funding> = app
            .wrap()
            .query_wasm_smart(
                &contracts.contract_incentives_addr,
                &QueryMsg::ProgramFunding { program_id: 1 },
            )
            .unwrap();
        println!("{:?}", funding);

        app.update_block(|block| {
            block.height += 6 // shift +1 because epoch can be processed after the epoch block
        });

        let epoch_info = process_epoch(&mut app, contracts.clone(), 1);

        println!("{:?}", epoch_info);

        // withdraw rewards for alice at epoch 1
        let alice_rewards =
            withdraw_rewards(&mut app, contracts.clone(), &alice, 1);

        // expected: 200ATOM coins
        assert_eq!(vec![Coin::new(200u128, "ATOM")], alice_rewards,);

        // add bob lock
        mint_and_lock(
            &mut app,
            contracts.clone(),
            &bob,
            &[Coin::new(200u128, "NIBI_LP")],
            300,
        );

        // bob qualifies for next epoch
        app.update_block(|block| {
            block.height += 5;
        });

        // process epoch
        let epoch_info = process_epoch(&mut app, contracts.clone(), 1);
        println!("{:?}", epoch_info);

        // withdraw rewards for alice at epoch 2
        let alice_balance =
            withdraw_rewards(&mut app, contracts.clone(), &alice, 1);

        // expected: 200 + 0.33*200 ATOM coins
        assert_eq!(
            vec![Coin::new((200u32 + 66u32).into(), "ATOM")],
            alice_balance,
        );

        // withdraw rewards for bob at epoch 2
        let bob_balance = withdraw_rewards(&mut app, contracts.clone(), &bob, 1);

        // expected: 200 * 0.66666666 ATOM
        assert_eq!(vec![Coin::new(133u128, "ATOM")], bob_balance,);

        app.update_block(|block| block.height += 1);

        // add more funding
        fund_program(
            &mut app,
            contracts.clone(),
            1,
            &[Coin::new(1000u128, "OSMO")],
        );

        // go to epoch 3 block and process it
        app.update_block(|block| block.height += 4);
        let epoch_info = process_epoch(&mut app, contracts.clone(), 1);
        println!("{:?}", epoch_info);

        // go to epoch 4 block and process it
        app.update_block(|block| block.height += 5);
        let epoch_info = process_epoch(&mut app, contracts.clone(), 1);
        println!("{:?}", epoch_info);

        // withdraw rewards for alice at epoch 3-4
        let alice_balance =
            withdraw_rewards(&mut app, contracts.clone(), &alice, 1);

        // withdraw rewards for bob at epoch 3-4
        let bob_balance = withdraw_rewards(&mut app, contracts.clone(), &bob, 1);

        assert_eq!(
            vec![Coin::new(399u128, "ATOM"), Coin::new(444u128, "OSMO")],
            bob_balance,
        );

        assert_eq!(
            vec![Coin::new(398u128, "ATOM"), Coin::new(222u128, "OSMO")],
            alice_balance,
        );

        // create a new funding for last epoch
        fund_program(
            &mut app,
            contracts.clone(),
            1,
            &[Coin::new(1000u128, "OSMO")],
        );
        // fast forward to epoch 5
        app.update_block(|block| block.height += 5);

        // process epoch 5
        app.update_block(|block| block.height += 5);
        let epoch_info = process_epoch(&mut app, contracts.clone(), 1);
        println!("{:?}", epoch_info);

        // finalize distribution
        let alice_balance =
            withdraw_rewards(&mut app, contracts.clone(), &alice, 1);

        assert_eq!(
            vec![Coin::new(464u128, "ATOM"), Coin::new(666u128, "OSMO")],
            alice_balance,
        );

        let bob_balance = withdraw_rewards(&mut app, contracts, &bob, 1);
        assert_eq!(
            vec![Coin::new(532u128, "ATOM"), Coin::new(1332u128, "OSMO")],
            bob_balance,
        );
        Ok(())
    }
}
