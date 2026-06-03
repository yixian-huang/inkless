export type LayoutType = "default" | "sidebar" | "fullwidth" | "landing" | "blank";

/** Content column profile — reading = narrow blog column, wide = marketing pages. */
export type ContentProfile = "reading" | "wide";

export interface LayoutConfig {
  type: LayoutType;
  contentProfile?: ContentProfile;
  header?: HeaderConfig;
  footer?: FooterConfig;
  sidebar?: SidebarConfig;
}

export interface HeaderConfig {
  style: "sticky" | "static" | "transparent";
  logo?: string;
  navigation?: NavItem[];
  showLanguageToggle?: boolean;
}

export interface FooterConfig {
  style: "full" | "minimal" | "none";
  logo?: string;
  address?: string;
  phone?: string;
  sections?: FooterSection[];
  copyright?: string;
}

export interface NavItem {
  label?: string;
  path?: string;
  children?: NavItem[];
}

export interface FooterSection {
  title?: string;
  links?: Array<{ label?: string; href?: string }>;
}

export interface SidebarConfig {
  position: "left" | "right";
  width?: string;
}
