use crate::document::Metadata;
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    env, fs,
    io::Write,
    path::{Path, PathBuf},
};

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
#[derive(Default)]
pub struct Config {
    pub theme: Theme,
    pub colors: Colors,
    pub display: Display,
    pub defaults: Defaults,
    pub recent: RecentConfig,
    pub versioning: Versioning,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct Theme {
    pub name: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct Display {
    pub check_symbol: String,
    pub select_marker: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct Defaults {
    pub file: String,
    pub max_visible: isize,
    pub word_wrap: bool,
    pub show_headings: bool,
    pub read_only: bool,
    pub filter_done: bool,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct RecentConfig {
    pub max_files: usize,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct Versioning {
    pub max_versions: i64,
}
impl Default for Theme {
    fn default() -> Self {
        Self {
            name: "tokyo-night".into(),
        }
    }
}
impl Default for Display {
    fn default() -> Self {
        Self {
            check_symbol: "✓".into(),
            select_marker: "➜".into(),
        }
    }
}
impl Default for Defaults {
    fn default() -> Self {
        Self {
            file: "todo.md".into(),
            max_visible: 0,
            word_wrap: true,
            show_headings: false,
            read_only: false,
            filter_done: false,
        }
    }
}
impl Default for RecentConfig {
    fn default() -> Self {
        Self { max_files: 20 }
    }
}
impl Default for Versioning {
    fn default() -> Self {
        Self { max_versions: 100 }
    }
}
pub fn home() -> Option<PathBuf> {
    env::var_os(if cfg!(windows) { "USERPROFILE" } else { "HOME" }).map(PathBuf::from)
}
pub fn directory() -> Result<PathBuf, String> {
    env::var_os("XDG_CONFIG_HOME")
        .filter(|v| !v.is_empty())
        .map(PathBuf::from)
        .or_else(|| home().map(|p| p.join(".config")))
        .map(|p| p.join("tdx"))
        .ok_or("cannot locate configuration directory".into())
}
pub fn resolve(path: &Path) -> Result<PathBuf, String> {
    let path = if let Some(tail) = path.to_str().and_then(|s| s.strip_prefix("~/")) {
        home().ok_or("home directory unavailable")?.join(tail)
    } else {
        path.to_owned()
    };
    std::path::absolute(path).map_err(|e| e.to_string())
}
impl Config {
    pub fn load() -> Self {
        let mut paths = Vec::new();
        if let Ok(dir) = directory() {
            paths.push(dir.join("config.toml"));
        }
        if let Some(home) = home() {
            paths.push(home.join(".config/tdx/config.toml"));
            if cfg!(target_os = "macos") {
                paths.push(home.join("Library/Application Support/tdx/config.toml"));
            }
        }
        if cfg!(windows)
            && let Some(appdata) = env::var_os("APPDATA")
        {
            paths.push(PathBuf::from(appdata).join("tdx/config.toml"));
        }
        let mut config = paths
            .iter()
            .find(|p| p.exists())
            .and_then(|p| fs::read_to_string(p).ok())
            .and_then(|s| toml::from_str::<Self>(&s).ok())
            .unwrap_or_default();
        if config.theme.name.is_empty() {
            config.theme = Theme::default();
        }
        if config.display.check_symbol.is_empty() {
            config.display.check_symbol = "✓".into();
        }
        if config.display.select_marker.is_empty() {
            config.display.select_marker = "➜".into();
        }
        if config.defaults.file.is_empty() {
            config.defaults.file = "todo.md".into();
        }
        if config.recent.max_files == 0 {
            config.recent.max_files = 20;
        }
        config.versioning.max_versions = config.versioning.max_versions.max(0);
        config
    }
}
#[derive(Clone, Debug, Default)]
pub struct Overrides {
    pub read_only: bool,
    pub show_headings: bool,
    pub max_visible: Option<isize>,
}
#[derive(Clone, Debug, Serialize)]
pub struct Settings {
    pub read_only: bool,
    pub show_headings: bool,
    pub max_visible: usize,
    pub word_wrap: bool,
    pub filter_done: bool,
}
impl Settings {
    pub fn new(config: &Config, meta: &Metadata, flags: &Overrides) -> Self {
        Self {
            read_only: flags.read_only || meta.read_only.unwrap_or(config.defaults.read_only),
            show_headings: flags.show_headings
                || meta.show_headings.unwrap_or(config.defaults.show_headings),
            max_visible: flags
                .max_visible
                .or(meta.max_visible)
                .unwrap_or(config.defaults.max_visible)
                .max(0) as usize,
            word_wrap: meta.word_wrap.unwrap_or(config.defaults.word_wrap),
            filter_done: meta.filter_done.unwrap_or(config.defaults.filter_done),
        }
    }
}
pub type Colors = BTreeMap<String, String>;
#[derive(Deserialize)]
struct ThemeFile {
    theme: ThemeIdentity,
    #[serde(default)]
    colors: Colors,
}
#[derive(Deserialize, Default)]
#[serde(default)]
struct ThemeIdentity {
    name: String,
}
pub fn themes() -> BTreeMap<String, Colors> {
    let mut result = BTreeMap::new();
    for source in BUILTIN_THEMES {
        if let Ok(t) = toml::from_str::<ThemeFile>(source) {
            result.insert(t.theme.name, t.colors);
        }
    }
    if let Ok(dir) = directory()
        && let Ok(entries) = fs::read_dir(dir.join("themes"))
    {
        let mut paths: Vec<_> = entries.filter_map(Result::ok).map(|e| e.path()).collect();
        paths.sort();
        for path in paths {
            if path.extension().is_some_and(|e| e == "toml")
                && let Ok(text) = fs::read_to_string(path)
                && let Ok(t) = toml::from_str::<ThemeFile>(&text)
                && !t.theme.name.is_empty()
            {
                result.insert(t.theme.name, t.colors);
            }
        }
    }
    result
}
pub fn save_theme_at(directory: &Path, name: &str) -> Result<(), String> {
    let path = directory.join("config.toml");
    let mut value = match fs::read_to_string(&path) {
        Ok(s) => toml::from_str::<toml::Value>(&s).map_err(|e| e.to_string())?,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            toml::Value::Table(Default::default())
        }
        Err(e) => return Err(e.to_string()),
    };
    let table = value
        .as_table_mut()
        .ok_or("configuration must be a table")?;
    let theme = table
        .entry("theme")
        .or_insert_with(|| toml::Value::Table(Default::default()))
        .as_table_mut()
        .ok_or("theme must be a table")?;
    theme.insert("name".into(), toml::Value::String(name.into()));
    fs::create_dir_all(directory).map_err(|e| e.to_string())?;
    let mut temp = tempfile::NamedTempFile::new_in(directory).map_err(|e| e.to_string())?;
    if let Ok(meta) = fs::metadata(&path) {
        temp.as_file()
            .set_permissions(meta.permissions())
            .map_err(|e| e.to_string())?;
    }
    temp.write_all(
        toml::to_string_pretty(&value)
            .map_err(|e| e.to_string())?
            .as_bytes(),
    )
    .map_err(|e| e.to_string())?;
    temp.as_file().sync_all().map_err(|e| e.to_string())?;
    temp.persist(path).map_err(|e| e.to_string())?;
    Ok(())
}

const BUILTIN_THEMES: &[&str] = &[
    include_str!("../../../cmd/tdx/themes/catppuccin-frappe.toml"),
    include_str!("../../../cmd/tdx/themes/catppuccin-latte.toml"),
    include_str!("../../../cmd/tdx/themes/catppuccin-macchiato.toml"),
    include_str!("../../../cmd/tdx/themes/catppuccin-mocha.toml"),
    include_str!("../../../cmd/tdx/themes/dracula.toml"),
    include_str!("../../../cmd/tdx/themes/github-dark.toml"),
    include_str!("../../../cmd/tdx/themes/gruvbox-dark.toml"),
    include_str!("../../../cmd/tdx/themes/monokai.toml"),
    include_str!("../../../cmd/tdx/themes/nord.toml"),
    include_str!("../../../cmd/tdx/themes/one-dark.toml"),
    include_str!("../../../cmd/tdx/themes/rose-pine.toml"),
    include_str!("../../../cmd/tdx/themes/solarized-dark.toml"),
    include_str!("../../../cmd/tdx/themes/tokyo-night.toml"),
];
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn theme_save_preserves_unknown_and_versioning() {
        let dir = tempfile::tempdir().unwrap();
        fs::write(
            dir.path().join("config.toml"),
            "[versioning]\nmax_versions=7\n[custom]\nkept=true\n[defaults]\nword_wrap=false\n",
        )
        .unwrap();
        save_theme_at(dir.path(), "nord").unwrap();
        let value: toml::Value =
            toml::from_str(&fs::read_to_string(dir.path().join("config.toml")).unwrap()).unwrap();
        assert_eq!(value["versioning"]["max_versions"].as_integer(), Some(7));
        assert_eq!(value["custom"]["kept"].as_bool(), Some(true));
        assert_eq!(value["defaults"]["word_wrap"].as_bool(), Some(false));
    }
    #[test]
    fn precedence_and_explicit_false() {
        let mut c = Config::default();
        c.defaults.read_only = true;
        let meta = Metadata {
            read_only: Some(false),
            word_wrap: Some(false),
            max_visible: Some(12),
            ..Metadata::default()
        };
        let flags = Overrides {
            read_only: true,
            max_visible: Some(3),
            ..Overrides::default()
        };
        let s = Settings::new(&c, &meta, &flags);
        assert!(s.read_only);
        assert!(!s.word_wrap);
        assert_eq!(s.max_visible, 3);
        assert!(!Settings::new(&c, &meta, &Overrides::default()).read_only);
        assert!(themes().len() >= 13);
    }
}
