#[test]
fn perm_mode_requires_the_policy_derive() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/*.rs");
}
