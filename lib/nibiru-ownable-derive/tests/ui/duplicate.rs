use nibiru_ownable_derive::PermPolicy;

#[derive(PermPolicy)]
enum Msg {
    #[perms(public)]
    #[perms(owner_or_any())]
    Duplicate,
}

fn main() {}
