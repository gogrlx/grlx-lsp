use std::{collections::HashMap, fs::canonicalize, path::PathBuf};

use anyhow::Result;
use lsp_server::{Connection, ExtractError, Message, Notification, Request, RequestId, Response};
use lsp_types::{
    request::GotoDefinition, Diagnostic, DiagnosticSeverity, DidChangeTextDocumentParams,
    DidOpenTextDocumentParams, GotoDefinitionResponse, InitializeParams, OneOf, Position, Range,
    ServerCapabilities, TextDocumentSyncKind, Url,
};
use nom::{
    bytes::complete::tag,
    character::complete::{alphanumeric1, char, multispace1, not_line_ending},
    combinator::opt,
    multi::count,
    sequence::{preceded, terminated},
    IResult,
};

/// Parse the includes from a grlx file
///
/// # Arguements
/// * `input` - The string to parse
/// # Returns
/// A HashMap of the line number and the path being referenced
fn parse_includes_map(input: &str, base_path: PathBuf) -> HashMap<usize, PathBuf> {
    input
        .lines()
        .enumerate()
        .skip_while(|&(_, line)| !line.starts_with("include:"))
        .take_while(|&(_, line)| !line.starts_with("steps:"))
        .filter_map(|(line_number, line)| {
            let line = parse_include_line(line);
            if let Ok(line) = line.as_ref() {
                let current = parse_current(line.1);
                if let Ok(current) = current.as_ref() {
                    if let Some(current) = current.1 {
                        let file = base_path.join(format!("{}.grlx", current));
                        return Some((line_number, file));
                    }
                }
            }
            line.ok()
                .map(|(_, path)| (line_number, base_path.join(format!("{}.grlx", path))))
        })
        .collect()
}

/// Parses a single line for an include
///
/// # Arguements
/// * `input` - The string to parse
/// # Returns
/// The path being referenced
fn parse_include_line(input: &str) -> IResult<&str, &str> {
    // remove the two spaces
    preceded(
        // This spcifically looks for 2 spaces
        count(char(' '), 2),
        preceded(
            // Look for a dash
            tag("-"),
            // remove the space after the dash, extract the rest of the line
            preceded(multispace1, nom::character::complete::not_line_ending),
        ),
    )(input)
}

/// Parses parses the case where a file might start with a dot
fn parse_current(input: &str) -> IResult<&str, Option<&str>> {
    opt(preceded(
        // Look for a dash
        char('.'),
        terminated(alphanumeric1, not_line_ending),
    ))(input)
}

/// Generates a diagnostic for a missing file
///
/// # Arguements
/// * `file_name` - The name of the file
/// * `line` - The line number the missing file is on
/// # Returns
/// A Diagnostic for the missing file
fn missing_file_diagnostic(file_name: PathBuf, line: u32) -> Diagnostic {
    let name: String = file_name
        .file_name()
        .and_then(|name| name.to_str())
        .map(|name| name.to_string())
        .unwrap();
    Diagnostic {
        range: Range {
            start: Position {
                line,
                character: 200,
            },
            end: Position {
                line,
                character: 200,
            },
        },
        severity: Some(DiagnosticSeverity::ERROR),
        code: None,
        code_description: None,
        source: Some("grlx-lsp".to_string()),
        message: format!("File {} does not exist", name),
        related_information: None,
        tags: None,
        data: None,
    }
}

/// Generates diagnostics for missing files
/// # Arguements
/// * `connection` - The connection to the client
/// * `files` - The files HashMap
/// * `file_name` - The name of the file
fn generate_diagnostics(
    connection: &Connection,
    files: &HashMap<String, HashMap<usize, PathBuf>>,
    file_name: Url,
) -> Result<()> {
    let name = file_name.to_string();
    let diagnostics = files
        .get(&name)
        .unwrap()
        .iter()
        .filter_map(|(line, file_name)| {
            if file_name.exists() {
                return None;
            }
            // let stem = PathBuf::from(file_name.file_stem().unwrap());
            // let dir = file_name.parent().unwrap().join(stem);
            // if dir.exists() {
            //     return None;
            // }
            Some(missing_file_diagnostic(file_name.clone(), *line as u32))
        })
        .collect::<Vec<_>>();
    if !diagnostics.is_empty() {
        let notification = lsp_types::PublishDiagnosticsParams {
            uri: file_name,
            diagnostics,
            version: None,
        };
        let notification = Notification {
            method: "textDocument/publishDiagnostics".to_string(),
            params: serde_json::to_value(notification).unwrap(),
        };
        connection
            .sender
            .send(Message::Notification(notification))?;
    }
    Ok(())
}

/// Updates the files HashMap in-place with the new file information
///
/// # Arguements
/// * `files` - A mutable reference to the files HashMap
/// * `file_name` - The name of the file
/// * `file` - The contents of the file
fn update_files(
    files: &mut HashMap<String, HashMap<usize, PathBuf>>,
    file_name: Url,
    file: String,
) {
    let base = file_name
        .to_file_path()
        .unwrap()
        .parent()
        .unwrap()
        .to_path_buf();
    let includes = parse_includes_map(&file, base);
    files.insert(file_name.to_string(), includes);
}


fn event_loop(connection: Connection, params: serde_json::Value) -> Result<()> {
    // For some reason, we must parse the params to allow for exiting
    let _params: InitializeParams = serde_json::from_value(params).unwrap();
    eprintln!("Starting main loop");
    let mut files: HashMap<String, HashMap<usize, PathBuf>> = HashMap::new();
    for msg in &connection.receiver { eprintln!("Connection received message {:?}", msg); match msg {
            Message::Request(req) => {
                let method = req.method.as_str();
                match method {
                    "textDocument/definition" => {
                        match cast::<GotoDefinition>(req) {
                            Ok((id, params)) => {
                                eprintln!("Received goto definition request {:?}", params);
                                let current_file =
                                    params.text_document_position_params.text_document.uri;

                                if let Some(file) = files.get(&current_file.to_string()) {
                                    // let path = current_file.to_file_path().unwrap().parent().unwrap();
                                    let position = params.text_document_position_params.position;
                                    let line = position.line as usize;

                                    if let Some(mut path) = file.get(&line) {
                                        let dir_path = PathBuf::from(path.file_stem().unwrap());
                                        let dir_path =
                                            path.parent().unwrap().join(dir_path.clone());
                                        eprintln!("Dir Path: {}", dir_path.display());
                                        if !path.exists() {
                                            eprintln!("Path {} does not exist", path.display());
                                            if !dir_path.exists() {
                                                eprintln!(
                                                    "Dir Path {} does not exist",
                                                    dir_path.display()
                                                );
                                                continue;
                                            } else {
                                                path = &dir_path;
                                            }
                                        }
                                        // This is a case where we are actually referencing a file in the
                                        // same directory as the current file.
                                        let mut complete_path = canonicalize(path)?;
                                        if complete_path.is_dir() {
                                            complete_path = complete_path.join("init.grlx");
                                        }
                                        let final_path =
                                            lsp_types::Url::from_file_path(complete_path).unwrap();
                                        eprintln!("Path: {}", final_path);
                                        let result = GotoDefinitionResponse::Scalar(
                                            lsp_types::Location::new(
                                                final_path,
                                                lsp_types::Range::new(
                                                    lsp_types::Position::new(0, 0),
                                                    lsp_types::Position::new(0, 0),
                                                ),
                                            ),
                                        );
                                        let result = serde_json::to_value(&result).unwrap();
                                        let response = Response {
                                            id: id.clone(),
                                            result: Some(result),
                                            error: None,
                                        };
                                        connection.sender.send(Message::Response(response))?;
                                    }
                                }

                                continue;
                            }
                            Err(err @ ExtractError::JsonError { .. }) => panic!("{err:?}"),
                            Err(ExtractError::MethodMismatch(req)) => req,
                        };
                    }
                    _ => {}
                }
                // TODO: We need to handle multiple potential cases
            }
            Message::Response(resp) => {
                eprintln!("Received response {:?}", resp);
            }
            Message::Notification(resp) => {
                eprintln!("Received notification {:?}", resp);
                let value = resp.method.as_str();
                match value {
                    // TODO: Implement initial file opening logic
                    "textDocument/didOpen" => {
                        if let Ok(params) =
                            serde_json::from_value::<DidOpenTextDocumentParams>(resp.params)
                        {
                            let file = params.text_document.text;
                            let file_name = params.text_document.uri;
                            update_files(&mut files, file_name.clone(), file.to_string());
                            generate_diagnostics(&connection, &files, file_name)?;
                        }
                    }
                    "textDocument/didChange" => {
                        if let Ok(params) =
                            serde_json::from_value::<DidChangeTextDocumentParams>(resp.params)
                        {
                            let changes = params.content_changes;
                            let file = &changes[0].text;
                            let file_name = params.text_document.uri;
                            update_files(&mut files, file_name.clone(), file.to_string());
                            generate_diagnostics(&connection, &files, file_name)?;
                        }
                    }
                    _ => {}
                }
            }
        };
    }
    Ok(())
}


fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    eprintln!("Starting grlx-lsp...");

    let (connection, io_threads) = Connection::stdio();
    let server_capabilities = serde_json::to_value(ServerCapabilities {
        text_document_sync: Some(lsp_types::TextDocumentSyncCapability::Kind(
            TextDocumentSyncKind::FULL,
        )),
        definition_provider: Some(OneOf::Left(true)),
        ..Default::default()
    })
    .unwrap();

    let init_params = connection.initialize(server_capabilities).unwrap();
    event_loop(connection, init_params)?;
    io_threads.join()?;

    eprintln!("Shutting down");
    Ok(())
}

fn cast<R>(req: Request) -> Result<(RequestId, R::Params), ExtractError<Request>>
where
    R: lsp_types::request::Request,
    R::Params: serde::de::DeserializeOwned,
{
    req.extract(R::METHOD)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_include_line_test() {
        let input = "  - .apache";
        let expected = ".apache";
        let result = parse_include_line(input);
        assert_eq!(result.unwrap().1, expected);
    }

    #[test]
    fn parse_current_test() {
        let input = ".apache";
        let expected = "apache";
        let result = parse_current(input);
        assert_eq!(result.unwrap().1.unwrap(), expected);

        let input = "../apache";
        let result = parse_current(input);
        assert!(result.unwrap().1.is_none());
    }
}
