# an open movie database
The OMDb API is a JSON API web service to obtain movie information, all content and images on the site are contributed and maintained by our users. 

#### Executing the migrations
migrate -path=./migrations -database=$OMDB_DB_DSN up

To open a connection to the DB and list the tables with the `\dt` meta command.
```
psql $OMDB_DB_DSN
```
Run the `\d` meta command on the movies table to see the structure of the table.
```
omdb-> \d movies
```
