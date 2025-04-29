Pre-requisites
* You need to have Postgres and Go installed in your machine. 

Database Preparation
* Create a database named ```gator``` using Postgres. 
* In the sql/schema directory, there are 5 migration files starting from 001 to 005.
* Use a migration tool like ```goose``` to run each migration in order.
* This will set up necessary tables in your "gator" database

Installation Steps
* Build the application first using ```go build -o gator```
* Move the resulting executable named gator to 
```$GOPATH/bin/gator``` for macOS or Linux and ```$GOPATH/bin/gator.exe``` for windows
* Now you can use ```gator``` in cmd to run the program
* Set up ```gatorconfig.json``` file in your root directory. The json file should have two attributes ```db_url``` and ```current_user_name```. ```db_url``` should be the url of the local database.

Running The Application
* ```gator register {username}``` will register a new user keep the user logged in.
* ```gator login {username}``` will log the user in.
* ```gator addfeed {feed_name} {feed_url}``` will add a feed.
* ```gator feeds``` will display all the feeds.
* ```gator agg``` will fetch and save all the posts from the saved feeds starting from the oldest one.
* ```gator follow {feed_url}``` will make the current logged in user follow the specific feed with the given url
* ```gator unfollow {feed_url}``` will make the current logged in user unfollow the specific feed with the given url
* ```gator browse {post_count}``` will display the posts of the feeds which the current user have followed. post_count is the number of post displayed and the default is 2.

