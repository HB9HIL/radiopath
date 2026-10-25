# Load terrain from S3

How to serve the elevation model from an S3-compatible bucket instead of a directory, and how to fill that bucket
with Copernicus GLO-30. The variables are listed in [Configuration](../reference/configuration.md).

With `RADIOPATH_DEM_S3_ENDPOINT` and `RADIOPATH_DEM_S3_BUCKET` set, tiles are
fetched on demand with `GET {endpoint}/{bucket}/{prefix}{band}/{name}.hgt`
(path style, AWS Signature V4 when keys are given) and kept in memory like
directory tiles. Keys are grouped by latitude band, `srtm1/N47/N47E009.hgt`,
so a bucket covering the world stays browsable; directories are flat.
Only GetObject is used, so a read-only policy on the bucket is enough.
`scripts/sync-dem-s3.sh` fills the bucket from Copernicus GLO-30 (public AWS
bucket `copernicus-dem-30m`, one ~40 MB COG per degree) using the S3 settings
in `.env`: each tile is warped onto the 1" grid with gdalwarp together with its
east, south and south-east neighbours, because the source has 3600 pixels per
degree and the 3601st row and column belong to the next tile. Needs curl and
gdalwarp. Resumable, skips sea tiles, `-f` overwrites existing tiles:

```sh
scripts/sync-dem-s3.sh 34 59 -11 44         # Europe: lat 34..59, lon -11..44, about 1200 tiles / 31 GB
scripts/sync-dem-s3.sh -j 4 -- -90 89 -180 179 # the world: about 26 000 tiles / 680 GB, 8 h with 4 workers
scripts/sync-dem-s3.sh -f N47E009 N47E008   # single tiles, replaced
```

The key prefix `srtm1/` is historical: the tiles are Copernicus data in the
SRTM file format.

Fetching a 26 MB tile takes a moment on first use; size `RADIOPATH_DEM_CACHE_TILES`
so the working area stays in memory. The readiness probe does not check S3.
