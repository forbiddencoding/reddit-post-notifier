# Reddit Post Notifier

**Reddit Post Notifier** allows users to create schedules to be notified via E-Mail with new Reddit Posts containing a
specific keyword from user specified subreddits and the ability to set additional filters. The project uses Temporal for
custom schedules and to assist in the interaction with the Reddit API (rate limiting and retries).

The application exposes a REST API for CRUD operations. A web frontend for your convenience can be
found [here](https://github.com/forbiddencoding/reddit-post-notifier-web).

## Getting Started

### Prerequisites

To get started with the project, you'll need to have the following installed on your machine:

* **Golang 1.26+**: The application is written in Go
* **Docker** and **Docker Compose**: Used to run Temporal and Databases

### Navigating the Codebase

The application is structured as a **modular monolith**, with its functionality divided into three services.
Each service can be run independently.

| Directory           | Description                                                                                                                                                                                        |
|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `services/app`      | This service provides the REST API that handles user requests for creating, reading, updating, and deleting schedules. It acts as the interface between the web frontend and the Temporal backend. |
| `services/digester` | A Temporal Worker that runs the Digest Workflow and its associated Activities. This is responsible for compiling the list of matching Reddit posts and sending the email notification.             |
| `services/redditor` | A Temporal Worker that runs the Post Workflow and its Activities. This worker is dedicated to interacting with the Reddit Data API and handles rate limiting to avoid hitting API call limits.     |

### Running the Application

#### 1. Clone the repository

```bash
git clone https://github.com/forbiddencoding/reddit-post-notifier.git
cd reddit-post-notifier
```

#### 2. Create a Reddit App

Login to your Reddit Account and create a Reddit App [here](https://old.reddit.com/prefs/apps).

* Give the application a name you like
* Select `script` for the type of application
* Set the `redirect uri` to `http://localhost` (The redirect uri itself does not matter, but it cannot be left blank and
  needs to be a valid formatted uri)
* Click on `create app` and make sure to store the ClientID and ClientSecret somewhere safe for the later steps

#### 3. Create a Google Mail App Password

Login to your Google Account and visit https://myaccount.google.com/apppasswords and enter an App-Name.
And click on `Create`. Make sure to store the App Password somewhere safe, you won't be able to see it again.

#### 4. Configure the Application

Create a `.env` file in the root of the repository based on the `.env.example` file and populate the missing keys.
Use the Reddit App Credentials and Google Mail App Password from steps 2. and 3.

* The `RPN_REDDIT_USERAGENT` value has to be formatted like this:
  `go:<GITHUB_URL_OF_THE_PROJECT>:v<SEMANTIC_VERSION> (by /u/<YOUR_REDDIT_USERNAME>)`
* The Google Mail App Password has to be entered without spaces into `RPN_MAILER_GMAIL_APP_PASSWORD`
* All example configurations are for local use with the Makefile.

#### 4. Start the Project

Start the docker services and application by running `make run` in your terminal

## API

Should you use the REST API directly, without the Web UI, the documentation can be found [here](docs/API.md).

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) and [NOTICE](NOTICE) files for
details.

<p align="center">Made with ❤️ in the Black Forest.</p>