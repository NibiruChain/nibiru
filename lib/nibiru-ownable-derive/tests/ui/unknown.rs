use nibiru_ownable_derive::PermPolicy;

#[derive(PermPolicy)]
enum Msg {
    #[perms(all_of("sai_oper"))]
    Unknown,
}

fn main() {}
