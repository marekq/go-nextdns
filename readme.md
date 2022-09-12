go-nextdns
==========

Retrieve NextDNS logs from your account using an API key. This project is experimental and may contain bugs. 

Follow these steps to get started:

- Copy 'example.env' to '.env' and fill in your NextDNS profile and API key.
- Run 'go run main.go -6h now' to retrieve DNS logs from the last 6 hours until now.
- The results will be written to 'output.log'. 
