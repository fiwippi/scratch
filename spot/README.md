
## spot

Extract data from Spotify. All output is JSON.

### FAQ

**Q: How do I list all my unfollowed artists by name?**

```sh
./spot artists unfollowed | jq '.[].name'
```

