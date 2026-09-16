use nibiru_ownable_derive::PermPolicy;

#[derive(PermPolicy)]
#[perms(namespace = "Admin")]
enum Msg {
    #[perms(owner_or_any())]
    OwnerOnly,
}

fn main() {}
