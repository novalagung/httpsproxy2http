# httpsproxy2http - Quick HTTPS reverse/forward proxy for your HTTP web service

When your site is running with HTTPS enabled, and tried to embed an URL or perform an API call towards external URL which is HTTP (not HTTPS), you will see the error in below:

> Mixed Content: The page at 'https://your-frontend.com/' was loaded over HTTPS, but requested an insecure resource 'http://your-webservice-api.com/v2/some/endpoint?param=1'.
> This request has been blocked; the content must be served over HTTPS.

It's mean that your API call or request is somehow blocked by the browser due to https://your-frontend.com/ was loaded using HTTPS-enabled but http://your-webservice-api.com is not. Trying to perform a call to a HTTP website from a webpage loaded via HTTPS is not allowed by browser, because it is insecure.

This simple service will help you to resolve that error. Simply change your URL from:

```
http://your-webservice-api.com/v2/some/endpoint?param=1
```

To:

```
https://httpsproxy2http.novalagung.com/your-webservice-api.com/v2/some/endpoint?param=1
```

In summary, use the https://httpsproxy2http.novalagung.com as the host of your destination URL, and put your actual URL as path next to it.

## Disclaimer

We do not store any of your data. Use at your own risk. It's recommended to host httpsproxy2http on your own cloud.

## Usage

```bash
https://httpsproxy2http.novalagung.com/<your-url>
https://httpsproxy2http.novalagung.com/your-webservice-api.com/v2/some/endpoint?param=1
```

## Install in your own cloud

It's possible to setup httpsproxy2http to your own cloud.

```bash
# clone our repo
git clone https://github.com/novalagung/httpsproxy2http.git

# go to the project directory
cd httpsproxy2http

# open docker-compose.yaml, adjust the HOST environment variable

# start the app
docker-compose up -d
```

You don't have to worry about setting up the HTTPS etc, we cover that for you.

## License

MIT License

## Author

Noval Agung Prayogo
