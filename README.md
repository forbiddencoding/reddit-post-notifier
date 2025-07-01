# Reddit Post Notifier

**Reddit Post Notifier** allows users to create schedules to be notified via E-Mail with new Reddit Posts containing a
specific keyword from user specified subreddits and the ability to set additional filters. The project uses Temporal for
the schedules and handle the interaction with the Reddit API (Rate Limiting). The application exposes a REST API for
CRUD operations. A web frontend for your convenience can be
found [here](https://github.com/forbiddencoding/reddit-post-notifier-web).

## Getting Started

### Prerequisites

To get started with the project, you'll need to have the following installed on your machine:

* **Golang 1.24+**: The application is written in Go
* **Docker** and **Docker Compose**: Used to run Temporal and Databases

### Navigating the Codebase

The application is structured as a **modular monolith**, with its functionality divided into three services.
Each service can be run independently.

| Directory           | Description                                                                                                                                                                                        |
|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `services/app`      | This service provides the REST API that handles user requests for creating, reading, updating, and deleting schedules. It acts as the interface between the web frontend and the Temporal backend. |
| `services/digester` | A Temporal Worker that runs the Digest Workflow and its associated Activities. This is responsible for compiling the list of matching Reddit posts and sending the email notification.             |
| `services/redditor` | A Temporal Worker that runs the Post Workflow and its Activities. This worker is dedicated to interacting with the Reddit Data API and handles rate limiting to avoid hitting API call limits.     |

### Supported Databases

The project natively supports the following databases:

* PostgreSQL v17+
* MySQL v8+
* SQLite

Older versions and other compatible SQL databases should work as well, but have not been tested.

> [!IMPORTANT]
> When using SQLite, all services have to run on the same host.

### Database Schema

The schema for the database has to be applied manually. The schema can be found in the `schema/sql/reddipostnotifier`
directory.

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

By default, the application expects a `config.yml` file in the `config/` directory. Copy the contents of the
`config/example.config.yml` file into a new `config.yml` file and populate the missing values with the Reddit App
Credentials and Google Mail App Password from steps 2. and 3.

* The `reddit.userAgent` value has to be formatted like this:
  `go:<GITHUB_URL_OF_THE_PROJECT>:v<SEMANTIC_VERSION> (by /u/<YOUR_REDDIT_USERNAME>)`
* The Google Mail App Password has to be entered without spaces
* Uncomment one of the three `persistence.driver` and `persistence.dsn` keys or create your own. All example
  configurations are for local use with one of the compose files.

#### 4. Start the Docker services

The project comes with two docker compose files:

* `compose.yml`: Uses Postgres for Temporal and is used for the Reddit Post Notifier
* `compose-mysql.yml` Uses MySQL for Temporal and is used for the Reddit Post Notifier

Your choice of these compose files should depend on the database chosen in the configuration in step 4. For SQLite
either compose file works.

Start the docker services by running `docker compose up -f <COMPOSE_FILE_NAME>.yml`

#### 5. Database Setup

Choose one database, according to your configuration from step 4:

##### Postgres

1. Create a database with its name according to your configuration from step 4.
2. Connect to the database and execute the `schema/sql/postgres/redditpostnotifier/schema.sql`

##### MySQL

1. Create a database with its name according to your configuration from step 4.
2. Connect to the database and execute the `schema/sql/mysql/redditpostnotifier/schema.sql`

##### SQLite

1. Create a database file (i.e. `data/sqlite/local.db`) with its path according to your configuration from step 4.
2. Connect to the database and execute the `schema/sql/sqlite/redditpostnotifier/schema.sql`

#### 6. Start the Reddit Post Notifier Services

From the root of the project run the following command:

```bash
go run cmd/server/main.go start
```

This will start all services with the config file being `config/config.yml`.

##### Flags

| Option               | Description                                                                         |
|----------------------|-------------------------------------------------------------------------------------|
| `--config` or `-c`   | Config file path relative to the application. Default is `./config/config.yml`.     |
| `--services` or `-s` | A comma separated list of service(s) to start. Default is `app,digest,reddit` (all) |

## API

Should you use the REST API directly, without the Web UI, the documentation can be found [here](docs/API.md).

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) and [NOTICE](NOTICE) files for
details.

<p align="center">Made with ❤️ in the Black Forest.</p>