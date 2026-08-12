import { useEffect, useRef } from "react";
import L from "leaflet";
import "leaflet/dist/leaflet.css";

// PositionMap — shared Leaflet-based map for the position-bearing
// panels (APRS station fixes, AIS vessel tracks, ADS-B aircraft,
// DSC distress alerts). One <PositionMap points=… /> per panel;
// the points array drives marker placement, the camera auto-fits
// to the data on every re-render.
//
// Tile source: OpenStreetMap standard via the public tile server
// at a.tile.openstreetmap.org. Per the OSM tile usage policy this
// is fine for a single self-hosted operator console (no commercial
// embedding, no proxying), but power users running large fleets
// of GopherTrunk daemons should configure their own tile cache
// (MapTiler / Mapbox / a local tileserver-gl) — this is a follow-
// up once the tile config surface exists.

export interface MapPoint {
  // Stable per-row identity. Used as React key in the marker
  // dictionary so re-renders patch markers in place rather than
  // tearing them down + re-creating them every poll.
  id: string;

  latitude: number;
  longitude: number;

  // Color category — one of the spec colours below. Maps to a
  // CircleMarker fillColor + outline.
  kind: "aprs" | "ais" | "adsb" | "dsc-distress" | "unit" | "default";

  // Tooltip body. Rendered as bold first line + optional
  // additional details lines.
  label: string;
  detail?: string;
}

interface PositionMapProps {
  points: MapPoint[];
  // Map height in CSS pixels. Defaults to 360 — enough to be
  // readable at a quick glance, short enough to leave the table
  // beneath visible above the fold.
  heightPx?: number;
  // Fixed center to fall back to when points is empty. Default
  // (37.5, -122.0) → SF Bay, a reasonable land-water mix.
  fallbackCenter?: [number, number];
  fallbackZoom?: number;
}

const KIND_COLOR: Record<MapPoint["kind"], string> = {
  aprs: "#3b82f6",          // blue — APRS station / Mic-E tracker
  ais: "#06b6d4",           // cyan — marine vessel
  adsb: "#a855f7",          // purple — aircraft
  "dsc-distress": "#ef4444",// red — distress alert
  unit: "#22c55e",          // green — trunked subscriber radio (LRRP/GPS)
  default: "#6b7280",       // grey — fallback
};

const KIND_RADIUS: Record<MapPoint["kind"], number> = {
  aprs: 5,
  ais: 5,
  adsb: 6,
  "dsc-distress": 8, // emphasise distress
  unit: 5,
  default: 4,
};

export function PositionMap({
  points,
  heightPx = 360,
  fallbackCenter = [37.5, -122.0],
  fallbackZoom = 8,
}: PositionMapProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markersRef = useRef<Map<string, L.CircleMarker>>(new Map());

  // Lazy-init the Leaflet map on first mount; tear down on
  // unmount. The OSM tile layer + attribution are bound here.
  useEffect(() => {
    if (!containerRef.current) return;
    if (mapRef.current) return;

    const map = L.map(containerRef.current, {
      center: fallbackCenter,
      zoom: fallbackZoom,
      zoomControl: true,
      attributionControl: true,
      // OSM tile servers are throttled per-IP; the daemon is a
      // single-user tool so this is fine, but prevent the user
      // from accidentally pulling thousands of tiles on a quick
      // pan by clamping max zoom.
      maxZoom: 18,
      minZoom: 2,
    });

    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "© OpenStreetMap contributors",
      maxZoom: 18,
    }).addTo(map);

    mapRef.current = map;
    return () => {
      map.remove();
      mapRef.current = null;
      markersRef.current.clear();
    };
    // fallbackCenter and fallbackZoom are constants on first
    // mount; intentionally not re-creating the map on every
    // re-render — the points-effect below handles update-only
    // changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync markers + auto-fit camera on every points update.
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    const markers = markersRef.current;
    const seen = new Set<string>();

    for (const p of points) {
      seen.add(p.id);
      const existing = markers.get(p.id);
      const latLng: L.LatLngExpression = [p.latitude, p.longitude];
      if (existing) {
        existing.setLatLng(latLng);
        existing.setStyle({ fillColor: KIND_COLOR[p.kind] });
        existing.bindTooltip(tooltipHTML(p), { direction: "top" });
      } else {
        const m = L.circleMarker(latLng, {
          radius: KIND_RADIUS[p.kind],
          fillColor: KIND_COLOR[p.kind],
          color: "#ffffff",
          weight: 1,
          opacity: 1,
          fillOpacity: 0.85,
        });
        m.bindTooltip(tooltipHTML(p), { direction: "top" });
        m.addTo(map);
        markers.set(p.id, m);
      }
    }

    // Remove markers no longer present.
    for (const [id, m] of markers.entries()) {
      if (!seen.has(id)) {
        m.remove();
        markers.delete(id);
      }
    }

    // Auto-fit camera to the active points, with a reasonable
    // pad so markers aren't pinned to the edges.
    if (points.length > 0) {
      const bounds = L.latLngBounds(
        points.map((p) => [p.latitude, p.longitude] as L.LatLngTuple),
      );
      map.fitBounds(bounds, { padding: [40, 40], maxZoom: 12 });
    }
  }, [points]);

  return (
    <div
      ref={containerRef}
      data-testid="position-map"
      className="rounded border border-border overflow-hidden"
      style={{ height: heightPx }}
    />
  );
}

function tooltipHTML(p: MapPoint): string {
  const labelEsc = escapeHTML(p.label);
  const detail = p.detail ? escapeHTML(p.detail) : "";
  return detail
    ? `<strong>${labelEsc}</strong><br/><span>${detail}</span>`
    : `<strong>${labelEsc}</strong>`;
}

function escapeHTML(s: string): string {
  return s.replace(/[&<>"']/g, (c) => {
    switch (c) {
      case "&": return "&amp;";
      case "<": return "&lt;";
      case ">": return "&gt;";
      case "\"": return "&quot;";
      case "'": return "&#39;";
    }
    return c;
  });
}
