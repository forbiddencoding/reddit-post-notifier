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

The app service provides a JSON REST API for managing schedules. It is recommended to use the Web based UI instead of
direct interactions via i.e. curl. You can find the Web based
UI [here](https://github.com/forbiddencoding/reddit-post-notifier-web).

### Schema

Conditions:

- At least one (1) and at most ten (10) subreddit can be added per schedule.
- `subreddit` is the name of the Subreddit without the /r prefix
- If `include_nsfw` is set to `true`, posts marked as NSFW will be in the mail, but thumbnails will be blurred. Defaults
  to `false`.
- Sort by `new` posts.
- If `restrict_subreddit` is set to true, only posts from this subreddit will be in the mail. Defaults to `true`
  (Recommended).
- `schedule` is a CRON expression for the schedule

```json
{
  "id": 7345454555745751040, // only on existing schedules
  "keyword": "Ahri",
  "subreddits": [
    {
      "id": 7345454555745751041, // only on exisitng subreddits in the schedule
      "subreddit": "AhriMains",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    },
    {
      "subreddit": "LeagueOfLegends",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    },
    {
      "subreddit": "LeagueOfMemes",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    }
  ],
  "schedule": "0 0 * * *",
  "recipients": [
    {
      "id": 7345454555745751042, // only on exisiting recipients in the schedule
      "address": "test@test.mail"
    }
  ]
}
```

### Operations

#### Create a Schedule

Creates a new schedule for a keyword. Schema conditions apply

| Method | Endpoint       | 
|--------|----------------|
| POST   | `/v1/schedule` |

Request Body:

```json
{
  "keyword": "Ahri",
  "subreddits": [
    {
      "subreddit": "AhriMains",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    },
    {
      "subreddit": "LeagueOfLegends",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    },
    {
      "subreddit": "LeagueOfMemes",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    }
  ],
  "schedule": "0 0 * * *",
  "recipients": [
    {
      "address": "test@test.mail"
    }
  ]
}
```

Response

```
HTTP/1.1 201 Created
{
  "id": 7345454555745751040
} 
```

#### Update a Schedule

Update a schedule. This is a PUT operation if a value in the database is not part of the PUT request it will be deleted.

| Method | Endpoint            |
|--------|---------------------|
| PUT    | `/v1/schedule/{id}` |

Request Body:

```json
{
  "id": 7345454555745751040,
  "keyword": "Ahri",
  "subreddits": [
    {
      "id": 7345454629720690688,
      "subreddit": "AhriMains",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": false
    },
    {
      "subreddit": "AuroraMains",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": true
    }
  ],
  "schedule": "0 0 * * *",
  "recipients": [
    {
      "id": 7345454715443875840,
      "address": "mail@test.mail"
    },
    {
      "address": "mail@mail.test"
    }
  ]
}
```

Response

```
HTTP/1.1 204 No Content
```

In this example:

- The AhriMains subreddit will be updated to not restrict posts to this subreddit (not recommended).
- The LeagueOfLegends and LeagueOfMemes subreddits will be removed
- The AuroraMains subreddit will be added
- The test@test.mail recipient will be updated to mail@test.test
- The mail@mail.test recipient will be added

#### Get a Schedule by its ID

Returns a schedule by its ID.

| Method | Endpoint            |
|--------|---------------------|
| GET    | `/v1/schedule/{id}` |

Response

```
HTTP/1.1 200 OK
{
  "id": 7345454555745751040,
  "keyword": "Ahri",
  "subreddits": [
    {
      "id": 7345454629720690688,
      "subreddit": "AhriMains",
      "include_nsfw": false,
      "sort": "new",
      "restrict_subreddit": false
    }
  ],
  "schedule": "0 0 * * *",
  "recipients": [
    {
      "id": 7345454715443875840,
      "address": "mail@test.mail"
    }
  ]
}
```

#### Delete a Schedule

Deletes a schedule by its ID.

| Method | Endpoint            |
|--------|---------------------|
| DELETE | `/v1/schedule/{id}` |

Response

```
HTTP/1.1 204 OK
```

#### List all Schedule

Get all created schedules (This operation does not support pagination).

| Method | Endpoint       |
|--------|----------------|
| GET    | `/v1/schedule` |

Response

```
HTTP/1.1 200 OK
{
  "schedules": [
    {
      "id": 7345454555745751040,
      "keyword": "Ahri",
      "subreddits": [
        {
          "id": 7345454629720690688,
          "subreddit": "AhriMains",
          "include_nsfw": false,
          "sort": "new",
          "restrict_subreddit": false
        }
      ],
      "schedule": "0 0 * * *",
      "recipients": [
        {
          "id": 7345454715443875840,
          "address": "mail@test.mail"
        }
      ]
    }
  ]
}
```

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) and [NOTICE](NOTICE) files for
details.

<p align="center">Made with ❤️ in the Black Forest.</p>