# HackLang Tools
HackLang tools that can run on Windows using Docker or WSL.

### Installation
Download `hh_client` binary from the latest release and put in a `%PATH%` directory.

### Usage:
Create a `.hhtools` files in the base directory of your project.
```json
{
  "provider": "docker",
  "image": "hhvm/hhvm:3.12.1"
}
```
You can also specify `wsl` as provider to use Windows Subsystem for Linux if you have installed HHVM there.

Now, you can run `hh_client` against your project and it will work as if you're running `hh_client` inside the container or Windows Subsystem for Linux.
