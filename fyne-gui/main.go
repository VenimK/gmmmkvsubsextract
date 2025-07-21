package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// TrackItem represents a subtitle track with UI elements
type TrackItem struct {
	Num        int
	Lang       string
	Codec      string
	Name       string
	State      string
	Check      *widget.Check
	Status     *widget.Label
	ConvertOCR *widget.Check // Option to convert PGS to SRT using OCR
}

// checkDependencies verifies if all required external tools are installed
func checkDependencies() map[string]bool {
	results := make(map[string]bool)
	
	// Check for mkvmerge
	mkvmergeCmd := exec.Command("mkvmerge", "--version")
	results["mkvmerge"] = mkvmergeCmd.Run() == nil
	
	// Check for mkvextract
	mkvextractCmd := exec.Command("mkvextract", "--version")
	results["mkvextract"] = mkvextractCmd.Run() == nil
	
	// Check for Deno
	denoCmd := exec.Command("deno", "--version")
	results["deno"] = denoCmd.Run() == nil
	
	// Check for Tesseract (optional, as it might be bundled with the script)
	tesseractCmd := exec.Command("tesseract", "--version")
	results["tesseract"] = tesseractCmd.Run() == nil
	
	return results
}

func main() {
	trackList := container.NewVBox()
	a := app.NewWithID("com.gmm.mkvsubsextract")
	w := a.NewWindow("GMM MKV Subtitles Extract (Fyne)")
	w.Resize(fyne.NewSize(800, 600))
	
	// Check dependencies at startup
	dependencyResults := checkDependencies()
	
	var mkvPath string
	var outDir string
	var trackItems []*TrackItem
	
	selectedFile := widget.NewLabel("No MKV file selected.")
	selectedDir := widget.NewLabel("No output directory selected.")
	result := widget.NewMultiLineEntry()
	result.SetPlaceHolder("Extraction results will appear here.")
	result.Wrapping = fyne.TextWrapBreak
	result.MultiLine = true
	// Make the result area larger to show more debug information
	resultScroll := container.NewScroll(result)
	resultScroll.SetMinSize(fyne.NewSize(780, 200))
	
	// Set up file drop handling
	w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		// Handle key events if needed
	})
	
	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) > 0 {
			filePath := uris[0].Path()
			fileExt := strings.ToLower(filepath.Ext(filePath))
			
			if fileExt == ".mkv" {
				// Handle MKV file drop
				mkvPath = filePath
				a.SendNotification(&fyne.Notification{
					Title:   "File Dropped",
					Content: "MKV file loaded: " + filepath.Base(filePath),
				})
				
				// Update UI
				selectedFile.SetText(mkvPath)
				
				// Set output directory to the same directory as the MKV file
				outDir = filepath.Dir(mkvPath)
				selectedDir.SetText(outDir)
				
				// Clear previous tracks
				trackItems = []*TrackItem{}
				trackList.Objects = nil
				trackList.Refresh()
				
				result.SetText("MKV file dropped and loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file.")
			} else {
				a.SendNotification(&fyne.Notification{
					Title:   "Invalid File",
					Content: "Please drop an MKV file only.",
				})
			}
		}
	})
	
	// Display dependency check results
	dependencyStatus := "System Dependency Check:\n"
	allDependenciesInstalled := true
	for tool, installed := range dependencyResults {
		status := "✅ Installed"
		if !installed {
			status = "❌ Not found"
			allDependenciesInstalled = false
		}
		dependencyStatus += fmt.Sprintf("- %s: %s\n", tool, status)
	}
	
	if !allDependenciesInstalled {
		dependencyStatus += "\n⚠️ Some required tools are missing. Please install them before using all features.\n"
	} else {
		dependencyStatus += "\n✅ All required tools are installed.\n"
	}
	
	result.SetText(dependencyStatus)

	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1
	progress.SetValue(0)

	currentTrackLabel := widget.NewLabel("")

	// Button to select MKV file
	fileBtn := widget.NewButton("Select MKV File (or Drag & Drop)", func() {
		// Create a file filter for MKV files
		filter := storage.NewExtensionFileFilter([]string{".mkv"})
		
		// Use custom dialog with filter
		fd := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				return
			}
			
			filePath := file.URI().Path()
			fileExt := strings.ToLower(filepath.Ext(filePath))
			
			// Double-check that it's an MKV file
			if fileExt != ".mkv" {
				dialog.ShowError(fmt.Errorf("Please select an MKV file only."), w)
				return
			}
			
			mkvPath = filePath
			selectedFile.SetText(mkvPath)
			
			// Set output directory to the same directory as the MKV file
			outDir = filepath.Dir(mkvPath)
			selectedDir.SetText(outDir)
			
			// Clear previous tracks
			trackItems = []*TrackItem{}
			trackList.Objects = nil
			trackList.Refresh()
			
			result.SetText("MKV file loaded. Output directory automatically set to MKV location. Click 'Load Tracks' to analyze the MKV file.")
		}, w)
		
		fd.SetFilter(filter)
		fd.Show()
	})

	// Button to select output directory (optional, as it's auto-set)
	dirBtn := widget.NewButton("Change Output Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			
			outDir = uri.Path()
			selectedDir.SetText(outDir)
		}, w)
	})

	// Button to load tracks from MKV file
	loadTracksBtn := widget.NewButton("Load Tracks", func() {
		if mkvPath == "" {
			dialog.ShowError(fmt.Errorf("Please select or drag & drop an MKV file first."), w)
			return
		}
		
		// Run mkvmerge to get track info
		cmd := exec.Command("mkvmerge", "-J", mkvPath)
		output, err := cmd.Output()
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error running mkvmerge: %v", err), w)
			return
		}
		
		// Parse JSON output
		var mkvInfo map[string]interface{}
		err = json.Unmarshal(output, &mkvInfo)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Error parsing mkvmerge output: %v", err), w)
			return
		}
		
		// Extract tracks
		tracks, ok := mkvInfo["tracks"].([]interface{})
		if !ok {
			dialog.ShowError(fmt.Errorf("No tracks found in MKV file."), w)
			return
		}
		
		// Clear previous tracks
		trackItems = []*TrackItem{}
		trackList.Objects = nil
		
		// Process subtitle tracks
		for _, track := range tracks {
			trackMap, ok := track.(map[string]interface{})
			if !ok {
				continue
			}
			
			// Check if this is a subtitle track
			trackType, ok := trackMap["type"].(string)
			if !ok || trackType != "subtitles" {
				continue
			}
			
			// Get track properties
			properties, ok := trackMap["properties"].(map[string]interface{})
			if !ok {
				continue
			}
			
			trackID := int(trackMap["id"].(float64))
			
			// Get language with nil check
			var trackLang string
			if properties != nil {
				if lang, ok := properties["language"].(string); ok {
					trackLang = lang
				} else {
					trackLang = "und" // undefined language code
				}
			} else {
				trackLang = "und" // undefined language code
			}
			
			trackCodec := trackMap["codec"].(string)
			
			// Get track name if available
			var trackName string
			if name, ok := properties["track_name"].(string); ok {
				trackName = name
			} else {
				trackName = ""
			}
			
			// Create UI elements for this track
			check := widget.NewCheck("", nil)
			check.SetChecked(true)
			status := widget.NewLabel("[ ]")
			
			// Create track item
			t := &TrackItem{
				Num:    trackID,
				Lang:   trackLang,
				Codec:  trackCodec,
				Name:   trackName,
				State:  "Pending",
				Check:  check,
				Status: status,
			}
			
			// Add OCR option for PGS subtitles
			if trackCodec == "hdmv_pgs_subtitle" || trackCodec == "HDMV PGS" {
				t.ConvertOCR = widget.NewCheck("", nil)
				t.ConvertOCR.SetChecked(true)
			}
			
			trackItems = append(trackItems, t)
			
			// Create row for this track
			trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", trackID, trackLang, trackCodec, trackName))
			
			var row *fyne.Container
			if t.ConvertOCR != nil {
				// For PGS subtitles, show OCR option
				ocrLabel := widget.NewLabel("Convert to SRT")
				row = container.NewHBox(check, status, trackInfo, t.ConvertOCR, ocrLabel)
			} else {
				// For other subtitle formats
				row = container.NewHBox(check, status, trackInfo)
			}
			
			trackList.Add(row)
		}
		trackList.Refresh()
		
		result.SetText("Tracks loaded. Select the tracks you want to extract, then click 'Start Extraction'")
	})

	// Button to start extraction of selected tracks
	startExtractBtn := widget.NewButton("Start Extraction", func() {
		if mkvPath == "" || outDir == "" {
			dialog.ShowError(fmt.Errorf("Please select both MKV file and output directory."), w)
			return
		}
		
		go func() {
			selected := []*TrackItem{}
			for _, t := range trackItems {
				if t.Check.Checked {
					selected = append(selected, t)
				}
			}
			if len(selected) == 0 {
				// Thread-safe UI update
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "No Tracks",
					Content: "No tracks selected.",
				})
				return
			}
			
			// Set up progress bar
			fyne.Do(func() {
				result.SetText("Extracting selected tracks...")
				progress.Max = float64(len(selected))
				progress.SetValue(0)
			})
			
			tracksDone := 0
			var output []byte
			var err error
			
			for i, t := range selected {
				// Update UI on main thread
				fyne.Do(func() {
					currentTrackLabel.SetText(fmt.Sprintf("Extracting track %d of %d: %s (%s) %s", i+1, len(selected), t.Lang, t.Codec, t.Name))
				})
				
				// Extract the subtitle track
				var outFile string
				
				// Get base filename without extension
				mkvBaseName := filepath.Base(mkvPath)
				mkvBaseName = strings.TrimSuffix(mkvBaseName, filepath.Ext(mkvBaseName))
				
				// Check if this is a PGS track with OCR conversion requested
				if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS") {
					// First extract as PGS
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n[DEBUG] Starting PGS extraction process")
					})
					tempPgsFile := fmt.Sprintf("%s.track%d_%s.sup", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT
					
					// Get absolute paths for extraction
					absPgsPath := filepath.Join(outDir, tempPgsFile)
					
					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting PGS track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== PGS Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("PGS file: %s\n", tempPgsFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absPgsPath))
					})
					
					// Extract PGS first - use full command for debugging
					cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, tempPgsFile)
					fyne.Do(func() {
						result.SetText(result.Text + "\nRunning: " + cmdStr)
					})
					
					// Create the command with proper arguments
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempPgsFile))
					cmd.Dir = outDir
					
					// Run the command and capture output
					output, err = cmd.CombinedOutput()
					
					// Debug output - show command result
					fyne.Do(func() {
						result.SetText(result.Text + "\nCommand output: " + string(output))
						if err != nil {
							result.SetText(result.Text + "\nError: " + err.Error())
						}
					})
					
					// Check if the file was created and has content
					pgsFilePath := filepath.Join(outDir, tempPgsFile)
					fileInfo, statErr := os.Stat(pgsFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty (0 bytes)")
					} else {
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted PGS file (%d bytes)", fileInfo.Size()))
						})
					}
					
					if err == nil {
						// Debug point after successful extraction
						// Create a progress spinner for the conversion process
						conversionSpinner := widget.NewProgressBarInfinite()
						conversionLabel := widget.NewLabel("Converting PGS to SRT...")
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						
						// Track conversion start time
						conversionStartTime := time.Now()
						
						// Create a ticker to update elapsed time - use 500ms for smoother updates
						ticker := time.NewTicker(500 * time.Millisecond)
						go func() {
							defer ticker.Stop()
							var lastText string
							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								newText := fmt.Sprintf("Elapsed: %s", elapsed)
								// Only update UI if text has changed to reduce UI operations
								if newText != lastText {
									lastText = newText
									fyne.Do(func() {
										elapsedLabel.SetText(newText)
									})
								}
							}
						}()
						
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] PGS extraction completed successfully, starting conversion process")
							
							// Show the conversion spinner and label
							currentTrackLabel.SetText("Converting PGS to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								conversionSpinner,
								elapsedLabel,
							))
							trackList.Refresh()
						})
						
						// Use the user's custom pgs-to-srt-2 tool with Deno
						pgsToSrtScript := "/Users/venimk/Downloads/pgs-to-srt-2/pgs-to-srt.js"
						// Define the path to the trained data file
						trainedDataPath := filepath.Join(filepath.Dir(pgsToSrtScript), "tessdata_fast", "eng.traineddata")
						
						// Get absolute paths for input and output
						absInputPath := filepath.Join(outDir, tempPgsFile)
						absOutputPath := filepath.Join(outDir, outFile)
						
						// Check if the script exists
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n\n[DEBUG] Checking if script exists at: %s", pgsToSrtScript))
						})
						
						if _, statErr := os.Stat(pgsToSrtScript); statErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Script NOT found: %v", statErr))
							})
							return
						}
						
						fyne.Do(func() {
							result.SetText(result.Text + "\n[DEBUG] Script found!")
						})
						
						// Test if Deno is working correctly
						fyne.Do(func() {
							result.SetText(result.Text + "\n[DEBUG] Running Deno version test...")
						})
						testCmd := exec.Command("deno", "--version")
						testOutput, testErr := testCmd.CombinedOutput()
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Deno Version Test ===\n")
							if testErr != nil {
								result.SetText(result.Text + fmt.Sprintf("Deno test error: %v\n", testErr))
							} else {
								result.SetText(result.Text + fmt.Sprintf("Deno version: %s\n", string(testOutput)))
							}
						})
						
						// Show detailed file information
						// Build text updates in memory before applying to UI
						textUpdate := fmt.Sprintf("\nInput SUP file: %s\nOutput SRT file: %s\nTessdata file: %s\n", 
							absInputPath, absOutputPath, trainedDataPath)
						
						fyne.Do(func() {
							result.SetText(result.Text + textUpdate)
							
							// Check if input file exists and show size
							if fileInfo, err := os.Stat(absInputPath); err == nil {
								result.SetText(result.Text + fmt.Sprintf("Input file size: %d bytes\n", fileInfo.Size()))
							} else {
								result.SetText(result.Text + fmt.Sprintf("Input file check error: %v\n", err))
							}
						})
						
						// Variables to track file copy status
						var copyErr error
						var copySuccess bool
						
						// Create a temporary file for the output to avoid permission issues
						tmpOutputFile, tmpErr := os.CreateTemp("", "pgs_to_srt_*.srt")
						if tmpErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create temporary file: %v", tmpErr))
							})
							return
						}
						tmpOutputPath := tmpOutputFile.Name()
						tmpOutputFile.Close() // Close it so the script can write to it
						
						// Build and show the command - the script expects trained data path and input file, with output redirected
						cmdStr := fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"", pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath)
						// Combine text updates to reduce UI operations
						updateText := fmt.Sprintf("\n\n=== Executing Command ===\n%s\n\nConversion started at: %s\n", 
							cmdStr, time.Now().Format("15:04:05"))
						
						fyne.Do(func() {
							result.SetText(result.Text + updateText)
						})
						
						// Create a log file for real-time monitoring of the PGS to SRT conversion process
						logFileName := filepath.Join(outDir, fmt.Sprintf("%s.track%d_%s.conversion.log", mkvBaseName, t.Num, t.Lang))
						logFile, logErr := os.Create(logFileName)
						
						// Create a logger that will be used throughout this function
						var logger *log.Logger
						
						if logErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n⚠️ Could not create log file: %v", logErr))
							})
						} else {
							defer logFile.Close()
							logger = log.New(logFile, "", log.LstdFlags)
							logger.Printf("=== PGS to SRT Conversion Log ===\n")
							logger.Printf("Started at: %s\n", time.Now().Format("15:04:05"))
							logger.Printf("Input file: %s\n", absInputPath)
							logger.Printf("Final output file: %s\n", absOutputPath)
							logger.Printf("Temporary output file: %s\n", tmpOutputPath)
							logger.Printf("Script: %s\n", pgsToSrtScript)
							logger.Printf("Trained data: %s\n", trainedDataPath)
							logger.Printf("Working directory: %s\n", filepath.Dir(pgsToSrtScript))
							logger.Printf("PATH: %s\n\n", os.Getenv("PATH"))
							
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n📝 Created log file: %s", logFileName))
								result.SetText(result.Text + fmt.Sprintf("\n📂 Using temporary file: %s", tmpOutputPath))
							})
						}
						
						// Run the conversion tool with Deno - using shell to enable output redirection
						cmd = exec.Command("sh", "-c", fmt.Sprintf("deno run --allow-read --allow-write \"%s\" \"%s\" \"%s\" > \"%s\"", 
							pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
						
						// Set the working directory to ensure relative paths work correctly
						cmd.Dir = filepath.Dir(pgsToSrtScript)
						
						// Print the environment and command for debugging
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Environment ===\n")
							result.SetText(result.Text + fmt.Sprintf("Working directory: %s\n", cmd.Dir))
							result.SetText(result.Text + fmt.Sprintf("PATH: %s\n", os.Getenv("PATH")))
							result.SetText(result.Text + "\n=== Command ===\n")
							result.SetText(result.Text + fmt.Sprintf("deno run --allow-read --allow-write %s %s %s > %s\n", 
								pgsToSrtScript, trainedDataPath, absInputPath, tmpOutputPath))
						})
						
						// Set up pipes to capture output in real-time
						stdoutPipe, _ := cmd.StdoutPipe()
						stderrPipe, _ := cmd.StderrPipe()
						
						// Start the command
						startErr := cmd.Start()
						if startErr != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n\n❌ Failed to start command: %v", startErr))
							})
							if logFile != nil && logger != nil {
								logger.Printf("Failed to start command: %v\n", startErr)
							}
							err = startErr
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + "\n\n=== Starting Conversion Process ===\n")
								result.SetText(result.Text + "Check the log file for real-time output\n")
							})
							
							// Create a multi-writer to write to both the log file and capture the output
							var outputBuffer strings.Builder
							var stdoutWriter, stderrWriter io.Writer
							if logFile != nil && logger != nil {
								stdoutWriter = io.MultiWriter(logFile, &outputBuffer)
								stderrWriter = io.MultiWriter(logFile, &outputBuffer)
								logger.Printf("Command started successfully\n")
							} else {
								stdoutWriter = &outputBuffer
								stderrWriter = &outputBuffer
							}
							
							// Copy stdout and stderr to the writers in a buffered way to reduce UI updates
							go func() {
								bufReader := bufio.NewReaderSize(stdoutPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"
									if _, writeErr := stdoutWriter.Write([]byte(line)); writeErr != nil {
										break
									}
								}
							}()
							
							go func() {
								bufReader := bufio.NewReaderSize(stderrPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"
									if _, writeErr := stderrWriter.Write([]byte(line)); writeErr != nil {
										break
									}
								}
							}()
							
							// Wait for the command to complete
							err = cmd.Wait()
							output = []byte(outputBuffer.String())
							
							// Log the completion status
							if logFile != nil && logger != nil {
								if err != nil {
									logger.Printf("\n\nCommand completed with error: %v\n", err)
								} else {
									logger.Printf("\n\nCommand completed successfully\n")
								}
								logger.Printf("Finished at: %s\n", time.Now().Format("15:04:05"))
							}
							
							// Copy the temporary file to the final destination regardless of command success/failure
							// This allows us to potentially recover partial conversions even if the command had issues
							
							// Check if the temporary file exists before attempting to copy
							if _, statErr := os.Stat(tmpOutputPath); statErr == nil {
								if logFile != nil && logger != nil {
									logger.Printf("Copying temporary file %s to final destination %s\n", tmpOutputPath, absOutputPath)
								}
								
								// Create the parent directory for the output file if it doesn't exist
								outputDir := filepath.Dir(absOutputPath)
								if mkdirErr := os.MkdirAll(outputDir, 0755); mkdirErr != nil {
									copyErr = fmt.Errorf("failed to create output directory: %v", mkdirErr)
									if logFile != nil && logger != nil {
										logger.Printf("Error creating output directory: %v\n", mkdirErr)
									}
								} else {
									// Read the temporary file
									tmpContent, readErr := os.ReadFile(tmpOutputPath)
									if readErr != nil {
										copyErr = fmt.Errorf("failed to read temporary file: %v", readErr)
										if logFile != nil && logger != nil {
											logger.Printf("Error reading temporary file: %v\n", readErr)
										}
									} else {
										// Write to the final destination
										writeErr := os.WriteFile(absOutputPath, tmpContent, 0644)
										if writeErr != nil {
											copyErr = fmt.Errorf("failed to write to final destination: %v", writeErr)
											if logFile != nil && logger != nil {
												logger.Printf("Error writing to final destination: %v\n", writeErr)
											}
										} else {
											copySuccess = true
											if logFile != nil && logger != nil {
												logger.Printf("Successfully copied temporary file to final destination\n")
											}
											
											// Clean up the temporary file
											removeErr := os.Remove(tmpOutputPath)
											if removeErr != nil && logFile != nil && logger != nil {
												logger.Printf("Warning: Could not remove temporary file: %v\n", removeErr)
											} else if logFile != nil && logger != nil {
												logger.Printf("Removed temporary file\n")
											}
										}
									}
								}
							} else {
								copyErr = fmt.Errorf("temporary file not found: %v", statErr)
								if logFile != nil && logger != nil {
									logger.Printf("Error: Temporary file not found: %v\n", statErr)
								}
							}
							
							// If the command succeeded but copy failed, update the error
							if err == nil && copyErr != nil {
								err = copyErr
							}
						}
						
						// Prepare output text in memory before updating UI
						var outputText strings.Builder
						outputText.WriteString("\nFull command output:\n")
						
						// Limit output size to prevent UI sluggishness with very large outputs
						outputStr := string(output)
						const maxOutputLen = 10000 // Limit output to 10K chars
						if len(outputStr) > maxOutputLen {
							outputText.WriteString(outputStr[:maxOutputLen])
							outputText.WriteString("\n... [Output truncated, full output in log file] ...")
						} else {
							outputText.WriteString(outputStr)
						}
						
						// Add error message if needed
						if err != nil {
							outputText.WriteString("\n\n❌ Command error: " + err.Error())
						}
						
						// Update UI in a single operation
						fyne.Do(func() {
							result.SetText(result.Text + outputText.String())
						})
						
						// Show output
						fyne.Do(func() {
							// Calculate total conversion time
							conversionTime := time.Since(conversionStartTime).Round(time.Second)
							
							// Update status based on success or failure
							if err != nil {
								currentTrackLabel.SetText(fmt.Sprintf("Conversion failed after %s", conversionTime))
							} else {
								currentTrackLabel.SetText(fmt.Sprintf("Conversion completed in %s", conversionTime))
							}
							progress.Show()
							
							// Stop the ticker by removing the spinner container
							// Find and remove the conversion spinner container
							for i, obj := range trackList.Objects {
								if box, ok := obj.(*fyne.Container); ok {
									for _, child := range box.Objects {
										if label, ok := child.(*widget.Label); ok && label.Text == "Converting PGS to SRT..." {
											trackList.Objects = append(trackList.Objects[:i], trackList.Objects[i+1:]...)
											break
										}
									}
								}
							}
							trackList.Refresh()
							
							result.SetText(result.Text + "\n\n=== Conversion Results ===\n")
							result.SetText(result.Text + "Completed at: " + time.Now().Format("15:04:05") + "\n")
							
							// Always show the full output for better debugging
							outputStr := string(output)
							result.SetText(result.Text + "\nFull output: \n" + outputStr + "\n")
							
							if err != nil {
								result.SetText(result.Text + "\n❌ Error: " + err.Error() + "\n")
							} else {
								result.SetText(result.Text + "\n✅ Command completed successfully\n")
								
								// Show file copy operation status
								result.SetText(result.Text + "\n=== File Operations ===\n")
								result.SetText(result.Text + fmt.Sprintf("✓ Temporary file created: %s\n", tmpOutputPath))
								if copySuccess {
									result.SetText(result.Text + fmt.Sprintf("✓ Copied to final destination: %s\n", absOutputPath))
									result.SetText(result.Text + "✓ Temporary file cleaned up\n")
								} else if copyErr != nil {
									result.SetText(result.Text + fmt.Sprintf("❌ Failed to copy to final destination: %v\n", copyErr))
								}
							}
							
							// Ensure the text area scrolls to the bottom to show the latest output
							result.CursorRow = len(strings.Split(result.Text, "\n")) - 1
						})
						
						// Check current directory for debugging
						currentDir, _ := os.Getwd()
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n=== Path Debugging ===\n")
							result.SetText(result.Text + fmt.Sprintf("Current working directory: %s\n", currentDir))
							result.SetText(result.Text + fmt.Sprintf("Looking for output file at: %s\n", absOutputPath))
						})
						
						// List files in output directory to see what was created
						files, _ := os.ReadDir(outDir)
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nFiles in output directory (%s):\n", outDir))
							for _, file := range files {
								result.SetText(result.Text + fmt.Sprintf("- %s\n", file.Name()))
							}
						})
						
						// Check if SRT file was created and show details
						if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
							fyne.Do(func() {
								result.SetText(result.Text + "\n✅ SRT file created successfully!")
								result.SetText(result.Text + fmt.Sprintf("\n   - Path: %s", absOutputPath))
								result.SetText(result.Text + fmt.Sprintf("\n   - Size: %d bytes", fileInfo.Size()))
								result.SetText(result.Text + fmt.Sprintf("\n   - Modified: %s", fileInfo.ModTime().Format("15:04:05")))
								
								// Try to count lines in SRT file
								if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
									lines := strings.Split(string(srtContent), "\n")
									result.SetText(result.Text + fmt.Sprintf("\n   - Lines: %d", len(lines)))
									
									// Count subtitle entries (every 4 lines is typically one subtitle)
									subtitleCount := (len(lines) + 3) / 4 // rough estimate
									result.SetText(result.Text + fmt.Sprintf("\n   - Estimated subtitles: ~%d", subtitleCount))
								}
							})
						} else {
							err = fmt.Errorf("SRT file was not created: %v", statErr)
							fyne.Do(func() {
								result.SetText(result.Text + "\n❌ Error: " + err.Error())
							})
						}
					}
				} else {
					// Normal extraction without conversion
					// Use proper file extension based on codec
					var fileExt string
					if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" {
						fileExt = "sup"
					} else if t.Codec == "subrip" || t.Codec == "SubRip" {
						fileExt = "srt"
					} else if t.Codec == "ass" || t.Codec == "ssa" || t.Codec == "ASS" || t.Codec == "SSA" {
						fileExt = "ass"
					} else if t.Codec == "vobsub" || t.Codec == "VobSub" {
						fileExt = "idx"
					} else {
						// Use lowercase codec name as fallback
						fileExt = strings.ToLower(t.Codec)
					}
					
					outFile = fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, t.Num, t.Lang, fileExt)
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, outFile))
					cmd.Dir = outDir
					output, err = cmd.CombinedOutput()
				}
				
				// Update UI on main thread
				fyne.Do(func() {
					if err != nil {
						t.State = "Error"
						t.Status.SetText(fmt.Sprintf("[!] Track %d: %s (%s) %s - Error", t.Num, t.Lang, t.Codec, t.Name))
						result.SetText(string(output) + "\nExtraction failed: " + err.Error())
					} else {
						t.State = "Done"
						t.Status.SetText(fmt.Sprintf("[✓] Track %d: %s (%s) %s - Done", t.Num, t.Lang, t.Codec, t.Name))
						progress.SetValue(float64(tracksDone + 1))
					}
					
					// Update track list
					trackList.Objects = nil
					for _, tt := range trackItems {
						trackInfo := widget.NewLabel(fmt.Sprintf("Track %d: %s (%s) %s", tt.Num, tt.Lang, tt.Codec, tt.Name))
						
						if tt.ConvertOCR != nil {
							// For PGS subtitles, show OCR option
							ocrLabel := widget.NewLabel("Convert to SRT")
							row := container.NewHBox(tt.Check, tt.Status, trackInfo, tt.ConvertOCR, ocrLabel)
							trackList.Add(row)
						} else {
							// For other subtitle formats
							row := container.NewHBox(tt.Check, tt.Status, trackInfo)
							trackList.Add(row)
						}
					}
					trackList.Refresh()
				})
				
				tracksDone++
			}
			
			// Final UI update on main thread
			fyne.Do(func() {
				currentTrackLabel.SetText("")
				if tracksDone == len(selected) {
					result.SetText("Extraction complete!")
					progress.SetValue(progress.Max)
				} else {
					result.SetText(fmt.Sprintf("Extraction stopped after %d of %d tracks", tracksDone, len(selected)))
				}
			})
		}()
	})

	// Create button row for better layout
	buttonRow := container.NewHBox(loadTracksBtn, startExtractBtn)
	
	// Use app.NewWithID for better performance and to avoid preferences API warnings
	// This was already set at the beginning of main()
	
	// Use a more efficient layout with container.NewBorder for better performance
	topContent := container.NewVBox(
		widget.NewLabel("GMM MKV Subtitles Extract (Fyne)"),
		fileBtn,
		selectedFile,
		dirBtn,
		selectedDir,
		buttonRow,
		currentTrackLabel,
		progress,
	)
	
	middleContent := container.NewVBox(
		widget.NewLabel("Subtitle Tracks:"),
		trackList,
	)
	
	bottomContent := container.NewVBox(
		widget.NewLabel("Results:"),
		resultScroll,
	)
	
	// Use Border layout for more efficient rendering
	w.SetContent(container.NewBorder(
		topContent,
		bottomContent,
		nil,
		nil,
		middleContent,
	))

	w.ShowAndRun()
}
