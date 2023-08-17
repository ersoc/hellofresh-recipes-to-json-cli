# hellofresh-recipes-to-json-cli

## Usage

```bash
$ go run main.go -output /home/chef/my_recipes -delay 3000 -tld com
```

### Flags

- **output**: Path to store the output. 
  - Default: `./output`
- **delay**: Delay per request in milliseconds. 
  - Default: `5000` (5 seconds)
- **tld**: Top level domain of the HelloFresh website. 
  - Default: `de`