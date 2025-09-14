# TAlias

This project creates a UI to quickly find and run custom commands, see prototype preview gif below.

### Build Script
- Clone project
- `cd` to directory with `main.go` and run:
```
go build -o /usr/local/bin/talias main.go
```
- Restart shell

### Shell Wrapper

The Go script runs in its own process and cannot run commands in the parent shell, to fix this you can add a shell script that executes talias then reads the output and then runs it:

```
# add to ~/.zshrc or ~/.bashrc
function nav() {
  dir=$(talias)   # Run your Go TUI app and capture its output
  if [ -n "$dir" ]; then
    cd "$dir" || return
  fi
}
```

Optionally you can build the app to any other directory and then update the shell script to point there instead, e.g. `dir=$(~/bin/talias)`. Also note that the name of the function above will be what is used to call the application.

### Set Options

Create `~/.talias/options.json` with the following config:

```
[
  {
    "title": "cd to Desktop",
    "details": "cd ~/Desktop",
    "path": "~/Desktop"
  },
  {
    "title": "cd to Documents",
    "details": "cd ~/Documents",
    "path": "~/Documents"
  },
  {
    "title": "cd to Downloads",
    "details": "cd ~/Downloads",
    "path": "~/Downloads"
  }
]
```

### Prototype

![](https://github.com/user-attachments/assets/04f1f0b0-1535-41b2-88c0-a11512eace22)


