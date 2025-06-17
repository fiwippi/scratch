## halo

Halo is a simple image tagger.

### FAQ

**Q: Is there a sample config?**

Yup, [here](config.toml).

**Q: How do I extract all uploaded photos?**

Note:

- All images are JPEG
- ULIDs are encoded as HEX vs. Base32 which is how the library encodes it

```console
> mkdir -p output

> sqlite3 store.db "SELECT writefile(printf('./output/%s.jpeg', hex(id)), data) FROM originals ORDER BY id ASC;"
```
