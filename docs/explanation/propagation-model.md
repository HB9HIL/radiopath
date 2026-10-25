# The propagation model

What Radiopath computes, what it assumes and where its limits are.

## Model notes

- ITM is valid for 20 MHz to 20 GHz, antenna heights 0.5 to 3000 m and paths
  up to 2000 km. Paths below 1 km produce a warning.
- Copernicus GLO-30 (and SRTM) is a surface model (includes trees and
  buildings) and ITM has no clutter model, so predictions in forests and cities are optimistic unless
  the canopy layer is configured (see [Tree canopy](#tree-canopy)).
- The ITM "propagation mode" is derived from smooth-earth horizons. An
  obstructed path inside the smooth-earth line-of-sight range still reports
  "line of sight"; the path loss and the Fresnel clearance show the obstruction.
- Fixed environment: continental temperate climate, N0 = 301, epsilon = 15,
  sigma = 0.005 S/m, 50 % time/location/situation.
- Terrain is sampled at the resolution the elevation model reports (30 m for
  1" tiles, 90 m for 3"), bilinearly interpolated, capped at 4000 samples per
  path.
- Links and coverage use different ITM variability modes. Both ends of a link
  are at known positions, so location variability is switched off and the result
  is given twice: the median and the level available 99 % of the time. Coverage
  uses the mobile mode at the median.
- Vegetation at the ends of a link comes from the canopy layer and ITU-R
  P.833; buildings and anything else stay manual in the extra path loss field.
  Rule of thumb without the layer: about 1 dB per metre of foliage at 5 to
  6 GHz, saturating near 30 to 40 dB behind woodland, a few dB at VHF.
- A link can store the level measured on the built link; the detail page shows
  measured minus predicted, which is the only honest way to learn how far the
  model is off in your terrain.
- Area coverage runs ITM point-to-point on every radial prefix (as Radio Mobile
  and SPLAT! do). Radials are spaced one pixel apart at the outer edge, between
  360 and 3600 of them, one sample per resolution step; pixels take the value of
  the nearest radial sample. Nearby small obstacles therefore cast thin radial
  shadows, which is inherent to the method. The raster is north-up
  equirectangular; its stored bounds place it on the map. "Compute" only queues
  a job in PostgreSQL; a background worker of any replica with
  `RADIOPATH_WORKERS` > 0 claims it (`FOR UPDATE SKIP LOCKED`, a job running for
  more than 15 minutes is taken over), and the page follows its state live.
  30 km at 100 m takes about 1 s, 60 km about 5 s on 10 cores; time grows with
  roughly (range / resolution)^3 and scales with CPU count (GOMAXPROCS, i.e. the
  container CPU limit in Kubernetes).

## Tree canopy

ITM has no vegetation. With a canopy height layer configured, a link reads the
tree top height along the path and adds, per end, the loss of ITU-R P.833-10
§2.1 for a terminal in woodland, `A = A_m (1 - exp(-d γ / A_m))`, where `d` is
the length of the direct ray inside the canopy from the antenna until it rises
above the trees. `γ` and `A_m` are read from the recommendation's Fig. 2 and
eq. (2) for mixed forest (about 1.1 dB/m and 52 dB at 5.7 GHz, 0.2 dB/m and
24 dB at 950 MHz). P.833 defines this loss as excess over free space and
diffraction, so ITM keeps running on the bare terrain; the tree tops are drawn
in the profile and enter the Fresnel clearance figure. Woods in the middle of
a path are drawn but add no loss, and the manual extra path loss field stays
for what the layer does not know. Without a layer nothing changes.

Two caveats. GLO-30, like SRTM, is a surface model: over dense forest the
elevation already contains part of the canopy, so trees on top are an upper bound until a real
terrain model is used. And the layer is a satellite estimate; the measured
level on a built link remains the check.

How to build and serve the layer: [Build the tree canopy layer](../how-to/build-the-canopy-layer.md).

## Antenna patterns

Antennas are optional. Without one, gain is the same in every direction; with
one, the azimuth pattern is subtracted from the gain in the direction that
matters — towards the other site for a link, per radial for a coverage. Import
is MSI Planet (`.msi`, `.pln`), the format antenna manufacturers ship: only the
horizontal block is read, values are attenuation in dB below the main lobe, and
a `GAIN` without a unit is taken as dBd. Files with 720 half-degree values are
resampled to one value per degree. Picking an antenna on a link or coverage
fills in its gain and locks the field; the stored gain is always the main lobe
gain, and the pattern only subtracts from it.

Without a file, an antenna can be described instead: name, gain, 3 dB beamwidth
and front-to-back ratio give the usual sector shape,
12 (angle / beamwidth)^2 dB capped at the front-to-back ratio; 360 degrees of
beamwidth is an omni. Enough for planning, not a replacement for the
manufacturer's pattern.

A link aims both antennas at the other site unless an azimuth is given; a
coverage always takes an explicit azimuth, 0 degrees is north. Elevation
patterns and mechanical downtilt are not modelled.
