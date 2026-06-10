import type { ReactNode } from "react";
import type { GTConfig } from "../api/types";
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
import { ToneOutSection } from "./ToneOut";
import { BroadcastSection } from "./Broadcast";
import { BasebandSection } from "./Baseband";
import { PagingSection } from "./Paging";
import { APRSSection } from "./APRS";
import { AISSection } from "./AIS";
import { DSCSection } from "./DSC";
import { MDC1200Section } from "./MDC1200";
import { ADSBSection } from "./ADSB";
import { M17Section } from "./M17";

export interface SectionDef {
  // key is the snake/lowercase section id used by the nav, the docs map,
  // and the validator (so the badge + Docs link resolve). Components patch
  // the draft with the capitalized Go field name internally.
  key: string;
  // cfgKey is the GTConfig (Go) field name this section edits, used to flag
  // "configured" sections in the nav (draft differs from defaults).
  cfgKey: keyof GTConfig;
  label: string;
  render: () => ReactNode;
}

export const SECTIONS: SectionDef[] = [
  { key: "trunking", cfgKey: "Trunking", label: "Trunking", render: () => <TrunkingSection /> },
  { key: "radioreference", cfgKey: "RadioReference", label: "RadioReference Login", render: () => <RadioReferenceSection /> },
  { key: "sdr", cfgKey: "SDR", label: "SDR", render: () => <SDRSection /> },
  { key: "api", cfgKey: "API", label: "API & Web", render: () => <APISection /> },
  { key: "scanner", cfgKey: "Scanner", label: "Scanner", render: () => <ScannerSection /> },
  { key: "audio", cfgKey: "Audio", label: "Audio", render: () => <AudioSection /> },
  { key: "recordings", cfgKey: "Recordings", label: "Recordings", render: () => <RecordingsSection /> },
  { key: "storage", cfgKey: "Storage", label: "Storage", render: () => <StorageSection /> },
  { key: "retention", cfgKey: "Retention", label: "Retention", render: () => <RetentionSection /> },
  { key: "metrics", cfgKey: "Metrics", label: "Metrics", render: () => <MetricsSection /> },
  { key: "log", cfgKey: "Log", label: "Logging", render: () => <LogSection /> },
  { key: "diagnostics", cfgKey: "Diagnostics", label: "Diagnostics", render: () => <DiagnosticsSection /> },
  { key: "web", cfgKey: "Web", label: "Web UI", render: () => <WebSection /> },
  { key: "tone_out", cfgKey: "ToneOut", label: "Tone-out", render: () => <ToneOutSection /> },
  { key: "broadcast", cfgKey: "Broadcast", label: "Broadcast", render: () => <BroadcastSection /> },
  { key: "baseband", cfgKey: "Baseband", label: "Baseband", render: () => <BasebandSection /> },
  { key: "paging", cfgKey: "Paging", label: "Paging", render: () => <PagingSection /> },
  { key: "aprs", cfgKey: "APRS", label: "APRS", render: () => <APRSSection /> },
  { key: "ais", cfgKey: "AIS", label: "AIS", render: () => <AISSection /> },
  { key: "dsc", cfgKey: "DSC", label: "DSC", render: () => <DSCSection /> },
  { key: "mdc1200", cfgKey: "MDC1200", label: "MDC1200", render: () => <MDC1200Section /> },
  { key: "adsb", cfgKey: "ADSB", label: "ADS-B", render: () => <ADSBSection /> },
  { key: "m17", cfgKey: "M17", label: "M17", render: () => <M17Section /> },
];
