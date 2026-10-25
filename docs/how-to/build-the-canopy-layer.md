# Build the tree canopy layer

How to produce and serve the canopy height tiles that add vegetation loss to links. Why and how the layer enters
the calculation is explained in [The propagation model](../explanation/propagation-model.md#tree-canopy). Point
`RADIOPATH_CANOPY_DIR` or `RADIOPATH_CANOPY_S3_PREFIX` at the result (see
[Configuration](../reference/configuration.md)).

Tiles are `.chm` files on the SRTM 1" grid: 3601 x 3601 nodes, one byte per
node with the canopy top height in metres, 0 for no trees, row 0 at the north
edge, `N47E009.chm`. A missing tile means no trees, so only tiles with forest
need to exist. Heights come from the ETH Global Canopy Height 2020 map (10 m,
CC BY 4.0; Lang, Jetz, Schindler, Wegner: *A high-resolution canopy height
model of the Earth*, Nature Ecology & Evolution 7, 2023). That model reports
heights on meadows and rock as well (11 to 19 m on the Säntis summit), so the
converter keeps them only where ESA WorldCover 2021 (10 m, CC BY 4.0, same
tiles and grid) has tree cover — majority class per 30 m cell, tallest tree
per cell. `scripts/sync-canopy-s3.sh` does both and needs curl, python3 and
GDAL — or a running podman machine, then it uses the `ghcr.io/osgeo/gdal`
image:

```sh
scripts/sync-canopy-s3.sh 34 59 -11 44   # same box as the DEM, about 16 GB
scripts/sync-canopy-s3.sh -j 6 -- -60 80 -180 179   # the world, about 170 GB; -j runs source tiles in parallel
scripts/sync-canopy-s3.sh N45E009        # one 3x3 degree source tile
```
