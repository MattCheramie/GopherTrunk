import type { ReactNode } from "react";
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
  label: string;
  render: () => ReactNode;
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
  { key: "tone_out", label: "Tone-out", render: () => <ToneOutSection /> },
  { key: "broadcast", label: "Broadcast", render: () => <BroadcastSection /> },
  { key: "baseband", label: "Baseband", render: () => <BasebandSection /> },
  { key: "paging", label: "Paging", render: () => <PagingSection /> },
  { key: "aprs", label: "APRS", render: () => <APRSSection /> },
  { key: "ais", label: "AIS", render: () => <AISSection /> },
  { key: "dsc", label: "DSC", render: () => <DSCSection /> },
  { key: "mdc1200", label: "MDC1200", render: () => <MDC1200Section /> },
  { key: "adsb", label: "ADS-B", render: () => <ADSBSection /> },
  { key: "m17", label: "M17", render: () => <M17Section /> },
];
