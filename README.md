# Go Chat

The Go application lives in `app/`. Its HTML frontend is in `app/web/index.html`
and is embedded into the binary at build time. 

Starting with Homebrew installed, run from the repository root:

```sh
brew install just 
just install # install go + download deps
just run
```

`just run` will start the server and open browser on http://localhost:8080.
