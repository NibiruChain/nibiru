---
order: 4
---

# Python SDK

Python-based client for interacting with the Nibiru blockchain. {synopsis}

The `nibiru` package allows you to index, query, and send transactions on the Nibiru Blockchain using Python. It provides access to market data for analysis, visualization, indicator development, algorithmic trading, strategy backtesting, bot programming, and related software engineering.

The package is intended to be used by coders, developers, technically-skilled traders and  data-scientists for building trading algorithms.

## User Guidelines

### Installation from `PyPI`

```bash
pip install nibiru  # requires Python 3.7
```

You may need to update `pip` to get this to run:

```bash
python -m pip install --upgrade pip
```

### Reference Links

- GitHub repo: [NibiruChain/py-sdk](https://github.com/NibiruChain/py-sdk)
- Published as [nibiru - PyPI package](https://pypi.org/project/nibiru/)

### Documentation Website

Documentation can be found here: [Nibiru-py documentation](https://nibiru-py.readthedocs.io/en/latest/index.html)

- Learn more about opening and managing your spot and perp positions [here](https://nibiru-py.readthedocs.io/en/latest/nibiru.sdks.tx.html#nibiru-sdks-tx-package)
- Learn about querying the chain using the Sdk [here](https://nibiru-py.readthedocs.io/en/latest/nibiru.clients.html#nibiru-clients-package)
---

## Development Guidelines

Our recommended setup is `pyenv` in combination with `poetry`.

- `pyenv` is a tool for installing and managing Python interpreters. This will let you seamlessly switch between Python versions.
- `poetry` is used for managing virtual environments, dependency resolution, package installations, package building, and package publishing.
- We assume you're on a Unix machine such as WSL2 Ubuntu, MacOS, or a common Linux distro.

Currently, `nibiru` is created with Python 3.9.13. It may be compatible with higher versions, but we only run end-to-end tests in 3.9.13.

### Setting up a professional dev environment with `pyenv` and `poetry`

#### Pyenv for managing multiple Python interpreters

If you're on MacOS or a common Linux distro, you can install `pyenv` with brew.

```bash
brew install pyenv
```

You'll then need to add the following snippet to your shell config, e.g. your `.bash_profile`, `.bashrc`, or `.zshrc`.

```bash
export PYENV_ROOT="$HOME/.pyenv"
export PATH="$PYENV_ROOT/bin:$PATH"
eval "$(pyenv init -)"
eval "$(pyenv init --path)"
```

After using `source` on your config or restarting the shell, you should have the `pyenv` root command.

The command use to install any version of python is `pyenv install`. Display additional info for this command with `pyenv install --help`.

```bash
pyenv install 3.9.13 # example for nibiru
```

Once you have a version installed, you can print out the versions on your machine with:

```bash
pyenv versions
```

```
# example output
  system
* 3.9.13 (set by /home/realu/.python-version)
  3.10.4
```

In this example, I have 2 different interpreters installed on my machine. The one with the `*` is currently set as my **global interpreter**. This is set manually using the `pyenv global` command.

```bash
pyenv global 3.10.4   # switches the global interpreter to 3.10.4
```

You can verify this works as expected using `python --version`. You may be familiar with using `python3` as the command instead of `python`. With `pyenv`, this is not necessary.

Additional usage and installation instructions can be found in the [pyenv repo](https://github.com/pyenv/pyenv).

### Installing `poetry` for dependency resolution and publishing packages

Reference: [Poetry docs](https://python-poetry.org/docs/)

Poetry can be installed with both `curl` and `pip`. We recommended using `curl` so that it will be global to your machine.

NOTE We highly, highly, highly recommend that you DO NOT use `brew` to install `poetry`.
If you use `brew`, it's going to install directly to your system, which prevents you from being able to leverage `pyenv` to seamlessly switch between Python interpreters.

```bash
# installation with pip: recommended option in tandem with pyenv
pip install poetry
```

```bash
# For UNIX systems - installation with curl
curl -sSL https://install.python-poetry.org/ | python -
```

After this installation command, add the `poetry` binary to the path in your shell config (if it's not done automatically).

```bash
export PATH=$PATH:$HOME/.poetry/bin
```

### Installing external dependencies

The `nibiru` project is defined by its `pyproject.toml`. At the root of the repo, simply call:

```bash
poetry install
```

This will resolve dependencies between each of the project's packages and install them into a virtual environment.

## Running tests

#### Setting environment variables

There's currently a "devnet" running in GCP that the CI workflows use. You can find these secrets at [this notion page](https://www.notion.so/nibiru/Resources-and-Repo-Configs-b31aa8074a2b419d80b0c946ed5efab0) if you have access to it or contact one of the `CODEOWNERS` (@Unique-Divine, @matthiasmatt, @nibiruheisenberg).
This is useful so that you can run every part of the package code without needing to visit other repositories.

Set up a `.env` file to set environment variables for the tests.
The variables used in the CI build can be found in the `env` section of the [`pytests.yml` workflow](.github/workflows/pytests.yml):

```yaml
jobs:
  tests:
    env:
      # https://www.notion.so/nibiru/Resources-and-Repo-Configs-b31aa8074a2b419d80b0c946ed5efab0
      CHAIN_ID: ${{ secrets.CHAIN_ID }}
      HOST: ${{ secrets.HOST }}
      VALIDATOR_MNEMONIC: ${{ secrets.VALIDATOR_MNEMONIC }}
      GRPC_PORT: ${{ secrets.GRPC_PORT }}
      LCD_PORT: ${{ secrets.LCD_PORT }}
```

You'll need an `.env` configuration like this.

```bash
# Example configuration for the Nibiry Python SDK
HOST="..."
VALIDATOR_MNEMONIC="..."
ORACLE_MNEMONIC="..."
GRPC_PORT="..."
LCD_PORT="..."
CHAIN_ID="..."
NETWORK_INSECURE=true
```

#### Running the tests with `poetry` + `pytest`

After following the instructions for setting up `poetry`, you can run the tests with `poetry run pytest`:

```bash
poetry run pytest -p no:warnings # silences warnings
```

#### (option B). Install the `nibiru` package with `pip`

  ```bash
  # from local
  # build and install
  pip install .

  # from local build
  pip uninstall nibiru
  pip install nibiru --no-index --find-links /path/to/nibiru/py-sdk/dist

  # from pypi
  pip uninstall nibiru
  pip install nibiru
  ```


## Makefile and Protocol Buffers

See the [NibiruChain/sdk-proto-gen repository](https://github.com/NibiruChain/sdk-proto-gen).
### Generating types with protobuf

The objective is to run `make proto-gen`, which simply executes `scripts/protocgen.sh`.

In order to do this, you'll need to install a few packages on your system.

```bash
python -m pip install --user grpcio-tools
pip install mypy-protobuf
```

If you get a permissions error such as

```
rm: cannot remove 'proto/proto/epochs/query.proto': Permission denied
```

call `sudo chown -R [USER-NAME] proto` using the name of user directory.
You can find the value for `[USER-NAME]` quickly by running `whoami`. In other words, this should work:

```bash
sudo chown -R $(whoami) proto
```

You're done generating types once you've successfully called

```bash
make proto-gen
poetry build # equivalently, you can run `python -m build`
```

## Linting

Enable git hook which will perform linting before each commit:

```bash
poetry run pre-commit install
```

This will keep your code clean.

## Gotchas

The `protobuf` package must be version 3.20.x or lower. Otherwise, the following error appears at runtime.

```
nibiru/clients/__init__.py:1: in <module>
    from nibiru.clients.dex import Dex  # noqa
nibiru/clients/dex.py:8: in <module>
    from nibiru.proto.dex.v1 import query_pb2 as dex_type
nibiru/proto/dex/v1/query_pb2.py:16: in <module>
    from google.api import annotations_pb2 as google_dot_api_dot_annotations__pb2
../../../anaconda3/envs/divine/lib/python3.9/site-packages/google/api/annotations_pb2.py:30: in <module>
    from google.api import http_pb2 as google_dot_api_dot_http__pb2
../../../anaconda3/envs/divine/lib/python3.9/site-packages/google/api/http_pb2.py:48: in <module>
    _descriptor.FieldDescriptor(
../../../anaconda3/envs/divine/lib/python3.9/site-packages/google/protobuf/descriptor.py:560: in __new__
    _message.Message._CheckCalledFromGeneratedFile()
E   TypeError: Descriptors cannot not be created directly.
E   If this call came from a _pb2.py file, your generated code is out of date and must be regenerated with protoc >= 3.19.0.
E   If you cannot immediately regenerate your protos, some other possible workarounds are:
E    1. Downgrade the protobuf package to 3.20.x or lower.
E    2. Set PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python (but this will use pure-Python parsing and will be much slower).
E
E   More information: https://developers.google.com/protocol-buffers/docs/news/2022-05-06#python-updates
```

## Usage instructions for publishing.

You specify updates to publish using the commit (or PR) title with `bump-[version-keyword]`.
For the `poetry version` command, ysing any bump rule with a valid semver string will change the version inside `pyproject.toml`. For example,

```
poetry version patch # moves from x.y.14 to x.y.15
poetry version minor # moves from x.5.z to x.6.0
poetry version major # moves from 3.y.z to 4.0.0
```
The list of bump rules includes:
patch, minor, major, prepatch, preminor, premajor, prerelease.

So the list of available keywords you an put in a PR includes
- `bump-patch`:
- `bump-patch`: 0.0.0 → 0.0.1
- `bump-minor`: 0.0.* → 0.1.0
- `bump-major`: 0.*.* → 1.0.0
- `bump-prepatch`: 0.0.0 → 0.0.1-alpha0
- `bump-prerelease`: equivalent to `bump-prepatch`
- `bump-preminor`: 0.0.* → 0.1.0-alpha0
- `bump-premajor`: 0.*.* → 1.0.0-alpha0

These guidelines are in the release.yml for future reference.
