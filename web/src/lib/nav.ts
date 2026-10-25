import ArrowLeftRight from '@lucide/svelte/icons/arrow-left-right';
import Antenna from '@lucide/svelte/icons/antenna';
import MapPin from '@lucide/svelte/icons/map-pin';
import Radar from '@lucide/svelte/icons/radar';

/** The sections of the app, for the header and the tab bar on phones. */
export const sections = [
	{ href: '/sites', label: 'Sites', icon: MapPin },
	{ href: '/links', label: 'Links', icon: ArrowLeftRight },
	{ href: '/coverages', label: 'Coverage', icon: Radar },
	{ href: '/antennas', label: 'Antennas', icon: Antenna }
];
