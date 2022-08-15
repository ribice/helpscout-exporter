# Help-Scout Exporter

We were migrating data from [Help Scout](https://www.helpscout.com/) to another service and Help Scout doesn't offer any 'Export' option out of the box. Instead, their MailBox API has to be used.

This project only supports exporting conversations + threads, as we needed only that, but extending it to other resources should be easy.

As the access token is a 'long-living' one (48h), I didn't implement token refresh.

The required flag is an access_token (`at`) acquired by invoking the Token (https://api.helpscout.net/v2/oauth2/token) API with either `authorization_code` or `refresh_token` grant.

