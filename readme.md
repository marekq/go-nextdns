go-nextdns
==========

Stream and download logs from your NextDNS account. This project is experimental and may contain bugs. 

In current form, it's useful for simple debugging and logging of DNS activity on your local network. You could run the `stream` command during debugging, or use the `download` command to download logs locally. 

## Getting started
 
Follow these steps to get started:

- Copy `example.env` to `.env` and fill in your NextDNS profile and API key. You can retrieve these from the [NextDNS portal](https://my.nextdns.io/d8c532/setup).
- Make sure you have `golang` installed locally. On Mac, you can install it using `brew install golang`. 
- Run `go build` to build the executable locally. 
- Finally, test using `./nextdns stream` to stream all logs.

The stream and download logs are stored in your local directory. You can monitor these in realtime running 'tail -f <logfile.log>'.

## Example commands for CLI:

### Stream all logs
- `./nextdns stream`

### Stream logs with a specific keyword
- `./nextdns stream coldstart.dev`

### Download logs from last 6 hours
- `./nextdns download -6h now`

### Download logs between a start and end date
- `./nextdns download 2022-09-01 2022-09-05`
