# How to use

Before running

```sh
make chaosnet
```

you need to log into our private Docker image registry in order to pull the private images. Go to <https://github.com/settings/tokens/new> and generate a new token with `read:packages` scope. Copy the access token to your clipboard.

Next, run

```sh
docker login ghcr.io
```

 and enter your GitHub username for the `username` field, and your personal access token for the password.

Now you can run

```sh
make chaosnet
```

## How to force pull images from the registry

By default, images won't re-fetch from upstream registries. To force a pull, you can run

```sh
make chaosnet-build
```

to force re-build and pull images.

## FAQ

### `make chaosnet` says that "Additional property name is not allowed"

Make sure to update your docker application to version 23.0.1