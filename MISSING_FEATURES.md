# Missing Features

## YAML configuration.
We need more precise configuration as an option. We should still accept env vars, but for a number of configuration options, this is required.

## Which proxies to trust?
We need to be able to set multiple IP ranges from which we trust to send us traffic, and only whitelist those. If we're behind a firewall that blocks incoming direct traffic, we may want to be able to trust all traffic.

## Different proxies have different headers that they use to send originating IP.
CloudFlare, CloudFront, etc. have different headers they use for this, and we only support the one.

## Multiple relay targets
We should support sending traffic to multiple relay targets.

## CloudFlare / CloudFront geolocation headers
We should be logging geolocation headers.

## The payload and storage structure 
There still seems to be a a score. We don't want scores. We just want to expose the raw results of tests.

This software is supposed to be modeled after MMORPG logic, where the client is as dumb as possible.

The JS should execute as a 
```JSON
{
    "results":{
        "sound_channels": 4,
        "gpu": "nvidia blah",
        "window_width": "1024",
        "window_height": "1024"
    }
}
```
We can do more nesting than this, however, this is more logical and easier to process. We don't want it to be obvious from somebody looking at the result that we're doing client side bot checking, it should look like very aggressive telemetry, possibly for market research.

## Without middleware, we don't need to allow customization of the collection endpoint
It should always be location.href

## Badly designed things
header_fingerprint doesn't really make sense, since the headers will be different each time.
payload_entropy <- This seems useless
request_size <- This is something I already know
user_agent_analysis <- I would rather save the useragent, so that others can process it, rather than processing it and then not saving it in it's entirety


## /px.gif
This should be used to detect ad block, primarily. Ad block is a good signal that I'm not dealing with a bot.

## Basic IP testing?
Should I do basic testing against each IP once to see if it has an open ports 22, 80, 443, 8080, or 8081?

## Reverse IP information
Reverse IP is one set of data that would be highly useful, if we stored up to the second level domain. (google.com)

208.79.209.138, for example, has a reverse IP that resolves to whatsmyip.org.
## WHOIS information
One of the ways I can enrich IP address information is by querying WHOIS servers.

Knowing that malicious traffic is coming from a specific 

208.79.209.138, for example, is assigned to the company named 'Macfixer (C08158750)'

Automated usage of this is likely to breach Terms of Use, trigger rate limits/blocks, or require a commercial agreement, but that information is **incredibly valuable** for security purposes.

Might need to integrate with multiple bulk whois providers like WhoisXMLAPI, DomainTools, IPinfo, etc.

For this information, we would want to have a direct SQL/NoSQL database for storing this information, and we would likely want multiple configurable integrations with each provider.

WhoisXMLAPI charges $30 for 2,000 requests on their monthly plan.

## IP reputation
Checking for the IP reputation is a good way to test and see if somebody is a bot.

## I need to test specific malicious headless browsers to look for identifying quirks.
HeadlessX, Lightpanda, Playwright-Stealth, undetected-chromedriver, puppeteer-extra-plugin-stealth

## Limited port checking
Check ports 22, 80, 443, and 8080 when a client connects.

This seems like an obvious thing to do to enrich the data, and I know there's at least thousands of servics that already do this.