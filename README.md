# buffalo-ocean

A plugin for [https://gobuffalo.io](https://gobuffalo.io) that makes deploying to DigitalOcean easier.

*It assumes you are using Docker to deploy*

**Notice:**

A 1GB DigitalOcean Standard Droplet will be created for you when using this plugin and being that DigitalOcean does charge for their services, hosting your site on this size droplet with them will cost you a $5 monthly fee. ([DigitalOcean Pricing](https://www.digitalocean.com/pricing/))

## Installation

```bash
$ go get -u -v github.com/wolves/buffalo-ocean
```

## Setup

```bash
buffalo ocean setup --app-name YOURAPP --key YOUR_DIGITAL_OCEAN_KEY
```

This command will setup and create a new DigitalOcean server droplet for you and deploy your app to it, based on your projects Dockerfile.


## Deploying

The initial `setup` command will do a deploy at the end, but anytime after that initial setup, you'll want to use the `buffalo heroku deploy` command to push a new version of your application, it'll even run your migrations for you as-long-as that step is provided in your projects Dockerfile.

```bash
$ buffalo ocean deploy --app-name YOURAPP
```

### Flags/Options

There are a lot of flags and options you can use to manage what/how you deploy to DigitalOcean. Use the `--help` flag to see a list of them all.

### Credits

- [Amber Framework](https://github.com/amberframework/amber) - Many structural flow ideas for creating this plugin came from here. As-well-as how to manage DigitalOcean from docker-machine
- [Buffalo-Heroku](https://github.com/markbates/buffalo-heroku) - Referenced regularly for buffalo integration and golang guidance
- [Buffalo](https://gobuffalo.io/) - The whole reason this plugin exists
- The Buffalo Slack Channel - Thank you to everyone for the helpful guidance along the way
