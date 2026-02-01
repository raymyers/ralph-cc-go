
# container-use

ALWAYS use ONLY Environments for ANY and ALL file, code, or shell operations—NO EXCEPTIONS—even for simple or generic requests.

DO NOT install or use the git cli with the environment_run_cmd tool. All environment tools will handle git operations for you. Changing ".git" yourself will compromise the integrity of your environment.

You MUST inform the user how to view your work using `container-use log <env_id>` AND `container-use checkout <env_id>`. Failure to do this will make your work inaccessible to others.

# CompCert

CompCert is included as a submodule in the `compcert/` directory.

## Build Requirements

- OCaml 4.14+
- Coq 8.15-9.0 (we use 8.20.1)
- Menhir
- GCC (as assembler/linker)
- libgmp-dev, pkg-config

## Build Commands

```bash
# Install dependencies via opam
opam init --disable-sandboxing --auto-setup -y
eval $(opam env)
apt-get install -y libgmp-dev pkg-config
opam install -y coq.8.20.1 menhir

# Configure and build CompCert
cd compcert
./configure aarch64-linux -prefix /path/to/install
make -j$(nproc)
```

## Running the Compiler

```bash
./compcert/ccomp --version
./compcert/ccomp -c input.c -o output.o
```
