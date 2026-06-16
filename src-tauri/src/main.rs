#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::{Deserialize, Serialize};
use tauri::Manager;
use std::{
    io::{BufRead, BufReader},
    process::{Child, Command, Stdio},
    sync::Mutex,
};

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct SidecarConfig {
    base_url: String,
    token: String,
}

#[derive(Debug, Deserialize)]
struct SidecarStartup {
    host: String,
    port: u16,
    token: String,
}

#[derive(Default)]
struct SidecarState {
    config: Mutex<Option<SidecarConfig>>,
    child: Mutex<Option<Child>>,
}

#[tauri::command]
fn sidecar_config(state: tauri::State<'_, SidecarState>) -> Result<SidecarConfig, String> {
    state
        .config
        .lock()
        .map_err(|err| err.to_string())?
        .clone()
        .ok_or_else(|| "sidecar is not ready".to_string())
}

fn main() {
    tauri::Builder::default()
        .manage(SidecarState::default())
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            let (config, child) = start_sidecar().map_err(|err| err.to_string())?;
            let state = app.state::<SidecarState>();
            *state.config.lock().map_err(|err| err.to_string())? = Some(config);
            *state.child.lock().map_err(|err| err.to_string())? = Some(child);
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![sidecar_config])
        .run(tauri::generate_context!())
        .expect("error while running Code Agent Manager desktop");
}

fn start_sidecar() -> Result<(SidecarConfig, Child), Box<dyn std::error::Error>> {
    let sidecar_name = if cfg!(windows) { "cam-sidecar.exe" } else { "cam-sidecar" };
    let current_exe = std::env::current_exe()?;
    let bundled = current_exe.with_file_name(sidecar_name);
    let mut command = if bundled.exists() {
        Command::new(bundled)
    } else {
        Command::new(sidecar_name)
    };
    let mut child = command
        .args(["--host", "127.0.0.1", "--port", "0"])
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()?;

    let stdout = child.stdout.take().ok_or("sidecar stdout unavailable")?;
    let mut reader = BufReader::new(stdout);
    let mut line = String::new();
    reader.read_line(&mut line)?;
    let startup: SidecarStartup = serde_json::from_str(line.trim())?;
    let config = SidecarConfig {
        base_url: format!("http://{}:{}", startup.host, startup.port),
        token: startup.token,
    };
    Ok((config, child))
}
