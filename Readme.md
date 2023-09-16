# hosts-filter

A small tool for filtering domains by its top level domain and response status code. If the domain contains the specified top level domain and respond with HTTP status code less than 500 (or 400 if application run with flag `--accept_4xx_errors=false`), it'll be considered a success.

## Run

Clone repo, install [Go](https://go.dev/dl/) and run:

```bash
go run main.go --input_file=domains.txt --domain_filter_file=domain_filter.txt
```

## Help

### Parameters:

 - `input_file` - file with domains line by line. Default `domains.txt`;
 - `domain_filter_file` - file with top level domains line by line. Every domain from input_file will failed if its top level domain not listed in domain_filter_file. Default `domain_filter.txt`;
 - `output_success_file` - verified domains will written into this file. Default `success.txt`;
 - `output_fails_file` - failed verification domains will written into this file. Default `fails.txt`;
 - workers_count - number of domains checkers working same time. Default `100`;
 - `accept_4xx_errors` - if `true`, domains with response code `4XX` will be considered a success and only domains with response code `5XX` will be fails. If `false`, all `4XX` and `5XX` codes will be considered an errors. Default `true`;
