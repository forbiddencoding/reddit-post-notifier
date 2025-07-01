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