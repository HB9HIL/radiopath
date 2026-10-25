import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import iconUrl from 'leaflet/dist/images/marker-icon.png';
import iconRetinaUrl from 'leaflet/dist/images/marker-icon-2x.png';
import shadowUrl from 'leaflet/dist/images/marker-shadow.png';

// Leaflet guesses the marker images from its CSS path, which the bundle breaks; give it the bundled ones.
L.Icon.Default.mergeOptions({ iconUrl, iconRetinaUrl, shadowUrl, imagePath: '' });

type Point = { name: string; lat: number; lon: number };

/** A map with the proxied OpenStreetMap tiles. */
export function baseMap(el: HTMLElement): L.Map {
	const m = L.map(el);
	L.tileLayer('/tiles/{z}/{x}/{y}.png', {
		maxZoom: 19,
		attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
	}).addTo(m);
	return m;
}

/** A map with one marker per site, zoomed to fit them or to Switzerland when there are none. */
export function sitesMap(el: HTMLElement, sites: Point[]): L.Map {
	const m = baseMap(el);
	const pts = sites.map((s) => {
		L.marker([s.lat, s.lon]).addTo(m).bindPopup(s.name);
		return L.latLng(s.lat, s.lon);
	});
	if (pts.length) m.fitBounds(L.latLngBounds(pts), { padding: [30, 30], maxZoom: 12 });
	else m.setView([47.0, 8.5], 7);
	return m;
}

/** A map to pick a position: a click or dragging the marker calls pick; place moves the marker to typed coordinates. */
export function sitePicker(el: HTMLElement, pick: (lat: number, lon: number) => void) {
	const m = baseMap(el);
	let marker: L.Marker | null = null;
	m.setView([47.0, 8.5], 7);

	/** Puts the marker at ll, optionally zooming in to it. */
	function place(ll: L.LatLngExpression, zoomTo: boolean) {
		if (marker) marker.setLatLng(ll);
		else {
			marker = L.marker(ll, { draggable: true }).addTo(m);
			marker.on('dragend', () => set(marker!.getLatLng()));
		}
		if (zoomTo) m.setView(ll, Math.max(m.getZoom() ?? 0, 12));
	}
	/** Reports a picked position with five decimals, about a metre. */
	function set(ll: L.LatLng) {
		pick(Number(ll.lat.toFixed(5)), Number(ll.lng.toFixed(5)));
		place(ll, false);
	}
	m.on('click', (e: L.LeafletMouseEvent) => set(e.latlng));
	return {
		place: (lat: number, lon: number) => place([lat, lon], true),
		remove: () => m.remove()
	};
}

/** A map with both ends of a link and the line between them. */
export function linkMap(el: HTMLElement, a: Point, b: Point): L.Map {
	const m = baseMap(el);
	L.marker([a.lat, a.lon]).addTo(m).bindPopup(a.name);
	L.marker([b.lat, b.lon]).addTo(m).bindPopup(b.name);
	L.polyline(
		[
			[a.lat, a.lon],
			[b.lat, b.lon]
		],
		{ className: 'link-path' }
	).addTo(m);
	m.fitBounds(
		[
			[a.lat, a.lon],
			[b.lat, b.lon]
		],
		{ padding: [30, 30] }
	);
	return m;
}
