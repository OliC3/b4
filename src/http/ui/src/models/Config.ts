export type FakingStrategy =
  | "ttl"
  | "pastseq"
  | "randseq"
  | "tcp_check"
  | "md5sum";
export enum FakingPayloadType {
  RANDOM = 0,
  CUSTOM = 1,
  DEFAULT = 2,
}
export interface IFaking {
  strategy: FakingStrategy;
  sni: boolean;
  ttl: number;
  seq_offset: number;
  sni_seq_length: number;
  sni_type: FakingPayloadType;
  custom_payload: string;
}

export type FragmentationStrategy = "tcp" | "ip" | "none";
export interface IFragmentation {
  strategy: FragmentationStrategy;
  sni_position: number;
  sni_reverse: boolean;
  middle_sni: boolean;
}

export enum LogLevel {
  ERROR = 0,
  INFO = 1,
  TRACE = 2,
  DEBUG = 3,
}
export interface ILogging {
  level: LogLevel;
  instaflush: boolean;
  syslog: boolean;
}

export interface IDomainConfig {
  sni_domains: string[];
  geosite_path: string;
  geoip_path: string;
  geosite_categories: string[];
  geoip_categories: string[];
}

export interface IDomainStatistics {
  manual_domains: number;
  geosite_domains: number;
  total_domains: number;
  category_breakdown?: Record<string, number>;
  geosite_available: boolean;
}

export interface ICategoryPreview {
  category: string;
  total_domains: number;
  preview_count: number;
  preview: string[];
}

export type UdpMode = "drop" | "fake";
export type UdpFilterQuicMode = "disabled" | "all" | "parse";
export type UdpFakingStrategy = "none" | "ttl" | "checksum";
export interface IUdpConfig {
  mode: UdpMode;
  fake_seq_length: number;
  fake_len: number;
  faking_strategy: UdpFakingStrategy;
  dport_min: number;
  dport_max: number;
  filter_quic: UdpFilterQuicMode;
}

export interface IB4Config {
  queue_start_num: number;
  threads: number;
  mark: number;
  conn_bytes_limit: number;
  seg2delay: number;
  use_gso: boolean;
  use_conntrack: boolean;
  skip_iptables: boolean;

  faking: IFaking;
  logging: ILogging;
  domains: IDomainConfig;
  fragmentation: IFragmentation;
  udp: IUdpConfig;
}

export default class B4Config implements IB4Config {
  queue_start_num: number = 537;
  threads: number = 4;
  mark: number = 32768;
  conn_bytes_limit: number = 19;
  seg2delay: number = 0;
  use_gso: boolean = true;
  use_conntrack: boolean = false;
  skip_iptables: boolean = false;

  logging: ILogging = {
    level: LogLevel.INFO,
    instaflush: false,
    syslog: false,
  };

  faking: IFaking = {
    strategy: "pastseq",
    sni: true,
    ttl: 8,
    seq_offset: 10000,
    sni_seq_length: 1,
    sni_type: FakingPayloadType.DEFAULT,
    custom_payload: "",
  };

  fragmentation: IFragmentation = {
    strategy: "tcp",
    sni_position: 5,
    sni_reverse: true,
    middle_sni: true,
  };

  domains: IDomainConfig = {
    sni_domains: [],
    geosite_path: "",
    geoip_path: "",
    geosite_categories: [],
    geoip_categories: [],
  };

  udp: IUdpConfig = {
    mode: "drop",
    fake_seq_length: 6,
    fake_len: 64,
    faking_strategy: "none",
    dport_min: 0,
    dport_max: 0,
    filter_quic: "parse",
  };
}
