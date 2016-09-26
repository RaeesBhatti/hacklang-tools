# HackLang Tools
HackLang tools that can be run on any platform including Windows using Docker or WSL

### Installation
Download `hh_client` binary from the latest release and put in a directory that is accessible by `$PATH`

### Usage:
Create a `.hhtools` files in the base directory of your project.
```json
{
  "provider": "docker",
  "image": "hhvm/hhvm:3.12.1"
}
```
You can also specify `wsl` as provider to use Windows Subsystem for Linux. If you have installed HHVM there.

Now, you can run `hh_client` against your project and it will work as if you're running `hh_client` inside the container or Bash for Windows.
