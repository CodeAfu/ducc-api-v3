
## TODO

- Character details list *(Not account bound)*

<div align="center">
  <img src=".github/images/preview1.png" alt="layout" />
  <img src=".github/images/preview2.png" alt="card" />
</div>


- https://genshin.jmp.blue/elements/pyro/icon


## Scraper

- [GET subreddit request](https://www.reddit.com/dev/api#GET_new)


## curl commands

```bash
# Build your struct from here
curl -H "User-Agent: ducc/0.1" "https://www.reddit.com/r/Genshin_Impact/new.json?limit=5" | jq '[.data.children[].data]'

# With pagination cursors 
curl -H "User-Agent: ducc/0.1" "https://www.reddit.com/r/Genshin_Impact/new.json?limit=5" | jq '{after: .data.after, before: .data.before, posts: [.data.children[].data]}'

# Get last 100 posts
curl -H "User-Agent: ducc/0.1" "https://www.reddit.com/r/Genshin_Impact/new.json?limit=100" | jq '[.data.children[].data | {author, title, permalink}]'

# Find author from results
curl -H "User-Agent: ducc/0.1" "https://www.reddit.com/r/Genshin_Impact/new.json?limit=200" | jq '[.data.children[].data | select(.author == "SF-Uberman") | {author, title, permalink}]'

# Query next page (get more than 100, its pagination)
curl -H "User-Agent: ducc/0.1" "https://www.reddit.com/r/Genshin_Impact/new.json?limit=100" | jq '.data.after'
curl -H "User-Agent: ducc/0.1" "https://www.reddit.com/r/Genshin_Impact/new.json?limit=100&after=t3_XXXXX" | jq '[.data.children[].data]'
```
