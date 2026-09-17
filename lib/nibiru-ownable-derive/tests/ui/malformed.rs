use nibiru_ownable_derive::PermPolicy;

#[derive(PermPolicy)]
enum Msg {
    #[perms(owner_or_any(sai_oper))]
    Malformed,
}

fn main() {}
