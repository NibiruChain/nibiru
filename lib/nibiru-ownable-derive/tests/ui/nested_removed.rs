use nibiru_ownable_derive::PermPolicy;

#[derive(PermPolicy)]
enum Msg {
    #[perms(nested)]
    Nested(Inner),
}

enum Inner {}

fn main() {}
