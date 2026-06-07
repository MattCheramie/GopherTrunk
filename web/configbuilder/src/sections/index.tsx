import type { ReactNode } from "react";
import type { GTConfig } from "../api/types";
import { GenericSection } from "./Generic";
import { SDRSection } from "./SDR";
import { TrunkingSection } from "./Trunking";
import {
  APISection,
  AudioSection,
  DiagnosticsSection,
  LogSection,
  MetricsSection,
  RadioReferenceSection,
  RecordingsSection,
  RetentionSection,
  ScannerSection,
  StorageSection,
  WebSection,
} from "./simple";

export interface SectionDef {
  key: string;
  label: string;
  render: () => ReactNode;
}

// Advanced sections without a bespoke form yet fall back to the JSON
// editor (still with instructions + docs + validation).
function generic(key: keyof GTConfig, label: string, instructions: string): SectionDef {
  return { key: String(key), label, render: () => <GenericSection sectionKey={key} title={label} instructions={instructions} /> };
}

export const SECTIONS: SectionDef[] = [
  { key: "trunking", label: "Trunking", render: () => <TrunkingSection /> },
  { key: "sdr", label: "SDR", render: () => <SDRSection /> },
  { key: "api", label: "API & Web", render: () => <APISection /> },
  { key: "scanner", label: "Scanner", render: () => <ScannerSection /> },
  { key: "audio", label: "Audio", render: () => <AudioSection /> },
  { key: "recordings", label: "Recordings", render: () => <RecordingsSection /> },
  { key: "storage", label: "Storage", render: () => <StorageSection /> },
  { key: "retention", label: "Retention", render: () => <RetentionSection /> },
  { key: "metrics", label: "Metrics", render: () => <MetricsSection /> },
  { key: "log", label: "Logging", render: () => <LogSection /> },
  { key: "diagnostics", label: "Diagnostics", render: () => <DiagnosticsSection /> },
  { key: "radioreference", label: "RadioReference", render: () => <RadioReferenceSection /> },
  { key: "web", label: "Web UI", render: () => <WebSection /> },
  generic("ToneOut", "Tone-out", "Two-tone / single-tone paging detection profiles."),
  generic("Broadcast", "Broadcast", "Outbound call streaming: Broadcastify, RdioScanner, OpenMHz, Icecast."),
  generic("Baseband", "Baseband", "IQ capture and replay for offline analysis."),
  generic("Paging", "Paging", "POCSAG / FLEX pager decoders."),
  generic("APRS", "APRS", "APRS / AX.25 AFSK receiver channels."),
  generic("AIS", "AIS", "Marine AIS GMSK receiver channels."),
  generic("DSC", "DSC", "Marine Digital Selective Calling receiver."),
  generic("MDC1200", "MDC1200", "Motorola MDC1200 FFSK signaling decoder."),
  generic("ADSB", "ADS-B", "ADS-B aircraft tracking."),
  generic("M17", "M17", "M17 digital-voice link-setup decoder."),
];
