## halo

Halo is a simple image tagger for Exif images.

### Install

Halo depends on [exiftool](https://linux.die.net/man/1/exiftool) to work, you can install it via:

1. `sudo apt install exiftool`

2. `brew install exiftool`

### FAQ

**Q: Is there a sample config?**

Yup, [here](config.toml).

**Q: How do I extract all uploaded photos?**

```console
> mkdir -p output

> sqlite3 store.db "SELECT writefile(printf('./output/%s.jpeg', timestamp), data) FROM photos ORDER BY id ASC;"
```
