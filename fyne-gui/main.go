package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
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

	// Check for ffmpeg
	// First try Homebrew path explicitly (preferred)
	homebrewPath := "/opt/homebrew/bin/ffmpeg"
	ffmpegFound := false

	// Debug output for ffmpeg detection
	fmt.Println("[DEBUG] Checking for ffmpeg...")

	// Simple file existence check for Homebrew ffmpeg
	if _, err := os.Stat(homebrewPath); err == nil {
		fmt.Println("[DEBUG] Homebrew ffmpeg exists at", homebrewPath)
		// Just check if file exists and is executable
		ffmpegFound = true
		fmt.Println("[DEBUG] Homebrew ffmpeg found")
	} else {
		fmt.Println("[DEBUG] Homebrew ffmpeg not found at", homebrewPath, "error:", err)

		// Try standard path using -h flag instead of --version
		fmt.Println("[DEBUG] Trying standard ffmpeg path")
		ffmpegCmd := exec.Command("ffmpeg", "-h")
		output, err := ffmpegCmd.CombinedOutput()
		ffmpegFound = err == nil && strings.Contains(string(output), "usage")
		fmt.Println("[DEBUG] Standard ffmpeg check result:", ffmpegFound)
		if err != nil {
			fmt.Println("[DEBUG] Standard ffmpeg error:", err)
		}

		// If still not found, try common Miniconda/Anaconda path
		if !ffmpegFound {
			// Get home directory
			homeDir, err := os.UserHomeDir()
			if err == nil {
				// Check Miniconda path
				minicondaPath := filepath.Join(homeDir, "miniconda3", "bin", "ffmpeg")
				if _, err := os.Stat(minicondaPath); err == nil {
					fmt.Println("[DEBUG] Miniconda ffmpeg exists at", minicondaPath)
					// Just check if file exists and is executable
					ffmpegFound = true
					fmt.Println("[DEBUG] Miniconda ffmpeg found")
				}

				// Also check Anaconda path if needed
				if !ffmpegFound {
					anacondaPath := filepath.Join(homeDir, "anaconda3", "bin", "ffmpeg")
					if _, err := os.Stat(anacondaPath); err == nil {
						fmt.Println("[DEBUG] Anaconda ffmpeg exists at", anacondaPath)
						// Just check if file exists and is executable
						ffmpegFound = true
						fmt.Println("[DEBUG] Anaconda ffmpeg found")
					}
				}
			}
		}
	}

	fmt.Println("[DEBUG] Final ffmpeg found status:", ffmpegFound)
	results["ffmpeg"] = ffmpegFound

	// Check for vobsub2srt binary
	fmt.Println("[DEBUG] Checking for vobsub2srt...")
	vobsub2srtPath := "/usr/local/bin/vobsub2srt"
	vobsub2srtFound := false
	
	// Check if vobsub2srt exists at the expected path
	if fileInfo, err := os.Stat(vobsub2srtPath); err == nil {
		fmt.Println("[DEBUG] vobsub2srt exists at", vobsub2srtPath)
		
		// Check if the file is executable (Unix-style permission check)
		perm := fileInfo.Mode().Perm()
		isExecutable := (perm & 0111) != 0 // Check if any execute bit is set
		
		fmt.Println("[DEBUG] vobsub2srt executable permission check:", isExecutable)
		
		if isExecutable {
			// Just verify the binary exists and is executable
			vobsub2srtFound = true
			fmt.Println("[DEBUG] vobsub2srt found and is executable")
		} else {
			fmt.Println("[DEBUG] vobsub2srt exists but is not executable")
		}
	} else {
		fmt.Println("[DEBUG] vobsub2srt not found at", vobsub2srtPath, "error:", err)
		
		// Try standard path using which command
		fmt.Println("[DEBUG] Trying to find vobsub2srt in PATH")
		whichCmd := exec.Command("which", "vobsub2srt")
		output, err := whichCmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			altPath := strings.TrimSpace(string(output))
			fmt.Println("[DEBUG] Found vobsub2srt at", altPath)
			
			// Check if the file exists and is executable
			if fileInfo, err := os.Stat(altPath); err == nil {
				// Check if the file is executable (Unix-style permission check)
				perm := fileInfo.Mode().Perm()
				isExecutable := (perm & 0111) != 0 // Check if any execute bit is set
				
				vobsub2srtFound = isExecutable
				fmt.Println("[DEBUG] vobsub2srt executable permission check:", isExecutable)
			}
		}
	}
	
	fmt.Println("[DEBUG] Final vobsub2srt found status:", vobsub2srtFound)
	results["vobsub2srt"] = vobsub2srtFound

	return results
}

func main() {
	trackList := container.NewVBox()
	// Create a scrollable container for the track list
	trackListScroll := container.NewScroll(trackList)
	// Set a minimum size for the track list scroll area to show more tracks
	trackListScroll.SetMinSize(fyne.NewSize(850, 250))

	// Create app with explicit ID and set metadata directly
	a := app.NewWithID("com.gmm.subtitleforge")
	a.SetIcon(theme.FileTextIcon())

	// Create main window with explicit name
	w := a.NewWindow("Subtitle Forge")
	// Set app metadata on window
	w.SetMaster()
	w.CenterOnScreen()
	w.Resize(fyne.NewSize(900, 700))

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
	
	// Track if vobsub2srt specifically is missing
	vobsub2srtMissing := false
	
	for tool, installed := range dependencyResults {
		status := "✅ Installed"
		if !installed {
			status = "❌ Not found"
			allDependenciesInstalled = false
			
			// Check if vobsub2srt is the missing tool
			if tool == "vobsub2srt" {
				vobsub2srtMissing = true
			}
		}
		dependencyStatus += fmt.Sprintf("- %s: %s\n", tool, status)
	}

	if !allDependenciesInstalled {
		dependencyStatus += "\n⚠️ Some required tools are missing. Please install them before using all features.\n"
	} else {
		dependencyStatus += "\n✅ All required tools are installed.\n"
	}

	result.SetText(dependencyStatus)
	
	// Create a container for dependency-related buttons
	dependencyButtons := container.NewVBox()
	
	// Add install button for vobsub2srt if it's missing
	if vobsub2srtMissing {
		installButton := widget.NewButton("Install VobSub2SRT", func() {
			// Show a confirmation dialog before proceeding
			dialog.ShowConfirm("Install VobSub2SRT", 
				"This will install VobSub2SRT from the leonard-slass fork.\n\nThe installation requires sudo privileges and will prompt for your password.\n\nDo you want to continue?", 
				func(confirmed bool) {
					if confirmed {
						// Create a progress dialog
						progress := dialog.NewProgress("Installing VobSub2SRT", "Preparing installation...", w)
						progress.Show()
						
						// Run the installation script in a goroutine
						go func() {
							// Get the script path relative to the executable
							execPath, err := os.Executable()
							if err != nil {
								fmt.Println("[ERROR] Failed to get executable path:", err)
								execPath = "."
							}
							execDir := filepath.Dir(execPath)
							scriptPath := filepath.Join(execDir, "install_vobsub2srt.sh")
							
							// Update progress
							progress.SetValue(0.1)
							
							// Run the installation script
							cmd := exec.Command("bash", scriptPath)
							output, err := cmd.CombinedOutput()
							
							// Hide the progress dialog
							progress.Hide()
							
							if err != nil {
								// Show error dialog
								dialog.ShowError(fmt.Errorf("Installation failed: %v\n\n%s", err, string(output)), w)
							} else {
								// Show success dialog
								dialog.ShowInformation("Installation Complete", "VobSub2SRT has been successfully installed.\n\nPlease restart the application to use the VobSub to SRT conversion feature.", w)
								
								// Update the dependency status
								dependencyResults = checkDependencies()
								
								// Update the status text
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
							}
						}()
					}
				}, w)
		})
		
		// Add the install button to the dependency buttons container
		dependencyButtons.Add(installButton)
	}

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

			// Add OCR option for PGS subtitles, ASS/SSA subtitles, and VobSub subtitles
			if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" ||
				strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") ||
				strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station") ||
				t.Codec == "vobsub" || t.Codec == "VobSub" {
				t.ConvertOCR = widget.NewCheck("", nil)
				t.ConvertOCR.SetChecked(true)
			} else {
				t.ConvertOCR = nil
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
						// Create a detailed progress bar for the conversion process
						conversionProgress := widget.NewProgressBar()
						conversionProgress.Min = 0
						conversionProgress.Max = 100 // Percentage-based progress
						conversionProgress.SetValue(0)

						conversionLabel := widget.NewLabel("Converting PGS to SRT...")
						statusLabel := widget.NewLabel("Initializing OCR process...")
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Estimated time remaining: calculating...")

						// Track conversion start time and progress data
						conversionStartTime := time.Now()
						var progressMutex sync.Mutex
						var progressData = struct {
							currentFrame int
							totalFrames  int
							frameRate    float64 // frames processed per second
							lastUpdate   time.Time
						}{
							currentFrame: 0,
							totalFrames:  0, // Will be updated when we parse output
							frameRate:    0,
							lastUpdate:   time.Now(),
						}

						// Create a ticker to update elapsed time and estimated remaining time
						ticker := time.NewTicker(500 * time.Millisecond)
						go func() {
							defer ticker.Stop()
							var lastElapsedText, lastRemainingText string

							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)

								// Calculate estimated time remaining
								progressMutex.Lock()
								currentFrame := progressData.currentFrame
								totalFrames := progressData.totalFrames
								frameRate := progressData.frameRate
								progressMutex.Unlock()

								var newRemainingText string
								var progressValue float64

								if totalFrames > 0 && currentFrame > 0 && frameRate > 0 {
									// Calculate percentage complete
									progressValue = float64(currentFrame) / float64(totalFrames) * 100

									// Calculate remaining time
									framesRemaining := totalFrames - currentFrame
									secondsRemaining := float64(framesRemaining) / frameRate
									remaining := time.Duration(secondsRemaining * float64(time.Second))
									remaining = remaining.Round(time.Second)

									newRemainingText = fmt.Sprintf("Estimated time remaining: %s", remaining)
								} else {
									newRemainingText = "Estimated time remaining: calculating..."
									progressValue = 0
								}

								// Only update UI if text has changed to reduce UI operations
								if newElapsedText != lastElapsedText || newRemainingText != lastRemainingText {
									lastElapsedText = newElapsedText
									lastRemainingText = newRemainingText

									fyne.Do(func() {
										elapsedLabel.SetText(newElapsedText)
										remainingLabel.SetText(newRemainingText)
										conversionProgress.SetValue(progressValue)
									})
								}
							}
						}()

						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] PGS extraction completed successfully, starting conversion process")

							// Show the conversion progress bar and labels
							currentTrackLabel.SetText("Converting PGS to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								statusLabel,
								conversionProgress,
								container.NewHBox(
									elapsedLabel,
									widget.NewLabel("|"),
									remainingLabel,
								),
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

							// Regular expressions to extract progress information from the output
							frameProgressRegex := regexp.MustCompile(`Processing frame (\d+)/(\d+)`)
							statusUpdateRegex := regexp.MustCompile(`Status: (.+)`)

							// Copy stdout and stderr to the writers in a buffered way to reduce UI updates
							go func() {
								bufReader := bufio.NewReaderSize(stdoutPipe, 4096) // Use larger buffer
								scanner := bufio.NewScanner(bufReader)
								for scanner.Scan() {
									line := scanner.Text() + "\n"

									// Check for progress information in the output
									if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
										// Extract current frame and total frames
										currentFrame := 0
										totalFrames := 0
										fmt.Sscanf(matches[1], "%d", &currentFrame)
										fmt.Sscanf(matches[2], "%d", &totalFrames)

										progressMutex.Lock()
										// Update progress data
										if progressData.totalFrames == 0 {
											progressData.totalFrames = totalFrames
										}

										// Calculate frame rate
										if progressData.currentFrame > 0 {
											timeDiff := time.Since(progressData.lastUpdate).Seconds()
											frameDiff := currentFrame - progressData.currentFrame
											if timeDiff > 0 && frameDiff > 0 {
												// Smooth the frame rate calculation with a weighted average
												newFrameRate := float64(frameDiff) / timeDiff
												if progressData.frameRate > 0 {
													// 70% old rate, 30% new rate for smoother estimates
													progressData.frameRate = progressData.frameRate*0.7 + newFrameRate*0.3
												} else {
													progressData.frameRate = newFrameRate
												}
											}
										}

										progressData.currentFrame = currentFrame
										progressData.lastUpdate = time.Now()
										progressMutex.Unlock()

										// Update status label
										percentComplete := float64(currentFrame) / float64(totalFrames) * 100
										fyne.Do(func() {
											statusLabel.SetText(fmt.Sprintf("Processing frame %d of %d (%.1f%%)",
												currentFrame, totalFrames, percentComplete))
										})
									} else if matches := statusUpdateRegex.FindStringSubmatch(line); len(matches) == 2 {
										// Update status message
										statusMsg := matches[1]
										fyne.Do(func() {
											statusLabel.SetText(statusMsg)
										})
									}

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

									// Also check stderr for progress information
									if matches := frameProgressRegex.FindStringSubmatch(line); len(matches) == 3 {
										// Process frame progress from stderr (same as stdout handler)
										currentFrame := 0
										totalFrames := 0
										fmt.Sscanf(matches[1], "%d", &currentFrame)
										fmt.Sscanf(matches[2], "%d", &totalFrames)

										progressMutex.Lock()
										// Update progress data
										if progressData.totalFrames == 0 {
											progressData.totalFrames = totalFrames
										}
										progressData.currentFrame = currentFrame
										progressMutex.Unlock()
									}

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
				} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "ssa") || strings.Contains(strings.ToLower(t.Codec), "substation") || strings.Contains(strings.ToLower(t.Codec), "sub station")) {
					// ASS/SSA to SRT conversion
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n[DEBUG] Starting ASS/SSA to SRT conversion process")
					})
					tempAssFile := fmt.Sprintf("%s.track%d_%s.ass", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT

					// Get absolute paths for extraction
					absAssPath := filepath.Join(outDir, tempAssFile)

					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting ASS/SSA track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== ASS/SSA Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("ASS/SSA file: %s\n", tempAssFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absAssPath))
					})

					// Extract ASS/SSA first - use full command for debugging
					cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, tempAssFile)
					fyne.Do(func() {
						result.SetText(result.Text + "\nRunning: " + cmdStr)
					})

					// Create the command with proper arguments
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, tempAssFile))
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
					assFilePath := filepath.Join(outDir, tempAssFile)
					fileInfo, statErr := os.Stat(assFilePath)
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
							result.SetText(result.Text + fmt.Sprintf("\nSuccessfully extracted ASS/SSA file (%d bytes)", fileInfo.Size()))
						})
					}

					if err == nil {
						// Create a progress bar for the conversion process
						conversionProgress := widget.NewProgressBar()
						conversionProgress.Min = 0
						conversionProgress.Max = 100
						conversionProgress.SetValue(0)

						conversionLabel := widget.NewLabel("Converting ASS/SSA to SRT...")
						statusLabel := widget.NewLabel("Processing ASS/SSA file...")
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Converting...")

						// Track conversion start time
						conversionStartTime := time.Now()

						// Create a ticker to update elapsed time
						ticker := time.NewTicker(500 * time.Millisecond)
						go func() {
							defer ticker.Stop()
							var lastElapsedText string

							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								newElapsedText := fmt.Sprintf("Elapsed: %s", elapsed)

								// Only update UI if text has changed
								if newElapsedText != lastElapsedText {
									lastElapsedText = newElapsedText
									fyne.Do(func() {
										elapsedLabel.SetText(newElapsedText)
										conversionProgress.SetValue(50) // Simple indeterminate progress
									})
								}
							}
						}()

						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] ASS/SSA extraction completed successfully, starting conversion process")

							// Show the conversion progress bar and labels
							currentTrackLabel.SetText("Converting ASS/SSA to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								statusLabel,
								conversionProgress,
								container.NewHBox(
									elapsedLabel,
									widget.NewLabel("|"),
									remainingLabel,
								),
							))
							trackList.Refresh()
						})

						// Get absolute paths for input and output
						absInputPath := filepath.Join(outDir, tempAssFile)
						absOutputPath := filepath.Join(outDir, outFile)

						// Use ffmpeg to convert ASS/SSA to SRT
						fyne.Do(func() {
							result.SetText(result.Text + "\n\n[DEBUG] Using ffmpeg to convert ASS/SSA to SRT")
							statusLabel.SetText("Running ffmpeg conversion...")
						})

						// Get ffmpeg path - prioritize Homebrew version
						ffmpegPath := "ffmpeg" // Default fallback path

						// First check Homebrew path (preferred)
						homebrewPath := "/opt/homebrew/bin/ffmpeg"
						if _, err := os.Stat(homebrewPath); err == nil {
							ffmpegPath = homebrewPath
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] Using Homebrew ffmpeg: " + homebrewPath)
							})
						} else {
							// If Homebrew not found, check Miniconda as fallback
							homeDir, err := os.UserHomeDir()
							if err == nil {
								minicondaPath := filepath.Join(homeDir, "miniconda3", "bin", "ffmpeg")
								if _, err := os.Stat(minicondaPath); err == nil {
									ffmpegPath = minicondaPath
									fyne.Do(func() {
										result.SetText(result.Text + "\n[DEBUG] Using Miniconda ffmpeg: " + minicondaPath)
									})
								}
							}
						}

						// Create the ffmpeg command with the appropriate path
						cmd = exec.Command(ffmpegPath, "-i", absInputPath, "-f", "srt", absOutputPath)
						cmd.Dir = outDir

						// Run the command and capture output
						output, err = cmd.CombinedOutput()

						// Stop the ticker
						ticker.Stop()

						// Update UI with results
						fyne.Do(func() {
							result.SetText(result.Text + "\nffmpeg output: " + string(output))

							if err != nil {
								result.SetText(result.Text + "\nError converting ASS/SSA to SRT: " + err.Error())
								statusLabel.SetText("Conversion failed!")
								conversionProgress.SetValue(0)
							} else {
								result.SetText(result.Text + "\nSuccessfully converted ASS/SSA to SRT")
								statusLabel.SetText("Conversion completed!")
								conversionProgress.SetValue(100)

								// Check if the output file was created
								if _, statErr := os.Stat(absOutputPath); statErr == nil {
									result.SetText(result.Text + fmt.Sprintf("\nSRT file created at: %s", absOutputPath))
								} else {
									result.SetText(result.Text + "\nWarning: Cannot find converted SRT file: " + statErr.Error())
								}
							}

							// Update elapsed time one last time
							elapsed := time.Since(conversionStartTime).Round(time.Second)
							elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
							remainingLabel.SetText("Completed")
						})
					}
				} else if t.ConvertOCR != nil && t.ConvertOCR.Checked && (t.Codec == "vobsub" || t.Codec == "VobSub") {
					// VobSub to SRT conversion
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n[DEBUG] Starting VobSub to SRT conversion process")
					})
					
					// For VobSub, we extract both .idx and .sub files
					// The .idx file is the main file that contains timing and positioning information
					// The .sub file contains the actual subtitle images
					idxFile := fmt.Sprintf("%s.track%d_%s.idx", mkvBaseName, t.Num, t.Lang)
					outFile = fmt.Sprintf("%s.track%d_%s.srt", mkvBaseName, t.Num, t.Lang) // Final output will be SRT
					
					// Get absolute paths for extraction
					absIdxPath := filepath.Join(outDir, idxFile)
					
					// Debug output
					fyne.Do(func() {
						currentTrackLabel.SetText(fmt.Sprintf("Extracting VobSub track %d...", t.Num))
						result.SetText(result.Text + "\n\n=== VobSub Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s)\n", t.Num, t.Lang))
						result.SetText(result.Text + fmt.Sprintf("Output directory: %s\n", outDir))
						result.SetText(result.Text + fmt.Sprintf("IDX file: %s\n", idxFile))
						result.SetText(result.Text + fmt.Sprintf("Absolute path: %s\n", absIdxPath))
					})
					
					// Extract VobSub first - use full command for debugging
					cmdStr := fmt.Sprintf("mkvextract tracks \"%s\" %d:\"%s\"", mkvPath, t.Num, idxFile)
					fyne.Do(func() {
						result.SetText(result.Text + "\nRunning: " + cmdStr)
					})
					
					// Create the command with proper arguments
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, idxFile))
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
					idxFilePath := filepath.Join(outDir, idxFile)
					fileInfo, statErr := os.Stat(idxFilePath)
					if statErr != nil {
						fyne.Do(func() {
							result.SetText(result.Text + "\nCannot find extracted file: " + statErr.Error())
						})
						err = statErr
					} else if fileInfo.Size() == 0 {
						fyne.Do(func() {
							result.SetText(result.Text + "\nExtracted file is empty (0 bytes)")
						})
						err = fmt.Errorf("extracted file is empty")
					} else {
						// File exists and has content, proceed with conversion
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\nIDX file extracted successfully (%d bytes)", fileInfo.Size()))
							result.SetText(result.Text + "\n\n=== VobSub to SRT Conversion ===\n")
						})
						
						// Create UI elements for conversion progress
						conversionStartTime := time.Now()
						conversionLabel := widget.NewLabel("Converting VobSub to SRT...")
						statusLabel := widget.NewLabel("Starting conversion...")
						conversionProgress := widget.NewProgressBar()
						elapsedLabel := widget.NewLabel("Elapsed: 0s")
						remainingLabel := widget.NewLabel("Estimating...")
						
						// Start a ticker to update the elapsed time
						ticker := time.NewTicker(time.Second)
						go func() {
							for range ticker.C {
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								fyne.Do(func() {
									elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
								})
							}
						}()
						
						// Show the conversion progress bar and labels
						fyne.Do(func() {
							currentTrackLabel.SetText("Converting VobSub to SRT...")
							progress.Hide()
							trackList.Add(container.NewVBox(
								conversionLabel,
								statusLabel,
								conversionProgress,
								container.NewHBox(
									elapsedLabel,
									widget.NewLabel("|"),
									remainingLabel,
								),
							))
							trackList.Refresh()
						})
						
						// Get absolute paths for input and output
						// For vobsub2srt, we need the base path without extension
						basePath := strings.TrimSuffix(idxFilePath, filepath.Ext(idxFilePath))
						absOutputPath := basePath + ".srt" // vobsub2srt will create this file
						
						// Check if both .idx and .sub files exist
						idxFile := basePath + ".idx"
						subFile := basePath + ".sub"
						
						fyne.Do(func() {
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for IDX file: %s", idxFile))
							result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Checking for SUB file: %s", subFile))
						})
						
						// Check if the files exist
						var filesExist bool = true
						if _, err := os.Stat(idxFile); err == nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file exists: %s", idxFile))
							})
						} else {
							filesExist = false
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] IDX file does not exist: %s - %v", idxFile, err))
							})
						}
						
						if _, err := os.Stat(subFile); err == nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file exists: %s", subFile))
							})
						} else {
							filesExist = false
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] SUB file does not exist: %s - %v", subFile, err))
							})
						}
						
						// If either file is missing, show a warning
						if !filesExist {
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] ⚠️ Warning: IDX or SUB file is missing, conversion may fail")
							})
						}
						
						// Pass the --lang parameter with the track language
						langCode := t.Lang
						if langCode == "" {
							langCode = "eng" // Default to English if no language code is available
						}
						
						// Map 3-letter language codes to 2-letter codes for vobsub2srt
						langCodeMap := map[string]string{
							"eng": "en", // English
							"fre": "fr", // French
							"fra": "fr", // French (alternate)
							"ger": "de", // German
							"deu": "de", // German (alternate)
							"ita": "it", // Italian
							"spa": "es", // Spanish
							"por": "pt", // Portuguese
							"dut": "nl", // Dutch
							"nld": "nl", // Dutch (alternate)
							"swe": "sv", // Swedish
							"nor": "no", // Norwegian
							"dan": "da", // Danish
							"fin": "fi", // Finnish
							"jpn": "ja", // Japanese
							"kor": "ko", // Korean
							"chi": "zh", // Chinese
							"zho": "zh", // Chinese (alternate)
							"rus": "ru", // Russian
							"pol": "pl", // Polish
							"cze": "cs", // Czech
							"ces": "cs", // Czech (alternate)
							"hun": "hu", // Hungarian
							"gre": "el", // Greek
							"ell": "el", // Greek (alternate)
							"tur": "tr", // Turkish
							"ara": "ar", // Arabic
							"heb": "he", // Hebrew
							"tha": "th", // Thai
						}
						
						// Convert 3-letter code to 2-letter code if a mapping exists
						if twoLetterCode, exists := langCodeMap[strings.ToLower(langCode)]; exists {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Mapped language code: %s -> %s", langCode, twoLetterCode))
							})
							langCode = twoLetterCode
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] No mapping found for language code: %s, using as-is", langCode))
							})
						}
						
						// Use vobsub2srt binary for conversion
						conversionScript := "/usr/local/bin/vobsub2srt"
						
						// Check if the binary exists
						if _, err := os.Stat(conversionScript); err != nil {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[ERROR] vobsub2srt binary not found at %s", conversionScript))
							})
							err = fmt.Errorf("vobsub2srt binary not found at %s", conversionScript)
						} else {
							fyne.Do(func() {
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using vobsub2srt binary: %s", conversionScript))
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Using language code: %s for VobSub conversion", langCode))
								result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Base path for vobsub2srt: %s", basePath))
							})
							
							// Check if the output SRT file already exists and delete it if it does
							outputSrtFile := basePath + ".srt"
							if _, err := os.Stat(outputSrtFile); err == nil {
								fyne.Do(func() {
									result.SetText(result.Text + fmt.Sprintf("\n[DEBUG] Removing existing SRT file: %s", outputSrtFile))
								})
								os.Remove(outputSrtFile)
							}
							
							// Run vobsub2srt with the language parameter
							cmdStr = fmt.Sprintf("%s --lang %s \"%s\"", conversionScript, langCode, basePath)
							fyne.Do(func() {
								result.SetText(result.Text + "\n[DEBUG] Running command: " + cmdStr)
								statusLabel.SetText("Running vobsub2srt conversion...")
							})
							
							// Create the command
							cmd = exec.Command(conversionScript, "--lang", langCode, basePath)
							cmd.Dir = outDir
							
							// Run the command and capture output
							output, err = cmd.CombinedOutput()
							
							// Stop the ticker
							ticker.Stop()
							
							// Update UI with results
							fyne.Do(func() {
								result.SetText(result.Text + "\nvobsub2srt output: " + string(output))
								
								if err != nil {
									result.SetText(result.Text + "\nError converting VobSub to SRT: " + err.Error())
									statusLabel.SetText("Conversion failed!")
									conversionProgress.SetValue(0)
								} else {
									result.SetText(result.Text + "\nSuccessfully ran vobsub2srt command")
									statusLabel.SetText("Conversion completed!")
									conversionProgress.SetValue(100)
									
									// Check if the output file was created
									if fileInfo, statErr := os.Stat(absOutputPath); statErr == nil {
										result.SetText(result.Text + fmt.Sprintf("\nSRT file created at: %s", absOutputPath))
										result.SetText(result.Text + fmt.Sprintf("\nSRT file size: %d bytes", fileInfo.Size()))
										
										// Try to count lines in SRT file
										if srtContent, readErr := os.ReadFile(absOutputPath); readErr == nil {
											lines := strings.Split(string(srtContent), "\n")
											result.SetText(result.Text + fmt.Sprintf("\nSRT file lines: %d", len(lines)))
											
											// Count subtitle entries (every 4 lines is typically one subtitle)
											subtitleCount := (len(lines) + 3) / 4 // rough estimate
											result.SetText(result.Text + fmt.Sprintf("\nEstimated subtitles: ~%d", subtitleCount))
										}
									} else {
										result.SetText(result.Text + "\nWarning: Cannot find converted SRT file: " + statErr.Error())
									}
								}
								
								// Update elapsed time one last time
								elapsed := time.Since(conversionStartTime).Round(time.Second)
								elapsedLabel.SetText(fmt.Sprintf("Elapsed: %s", elapsed))
								remainingLabel.SetText("Completed")
							})
						}
					}
				} else {
					// Normal extraction without conversion
					// Use proper file extension based on codec
					var fileExt string

					// Handle special case for "SubRip/SRT" format
					if strings.Contains(t.Codec, "SubRip") || strings.Contains(t.Codec, "subrip") || strings.Contains(t.Codec, "SRT") || strings.Contains(t.Codec, "srt") {
						fileExt = "srt"
						fyne.Do(func() {
							result.SetText(result.Text + "\nDetected SRT format, using .srt extension")
						})
					} else if t.Codec == "hdmv_pgs_subtitle" || t.Codec == "HDMV PGS" {
						fileExt = "sup"
					} else if t.Codec == "ass" || t.Codec == "ssa" || t.Codec == "ASS" || t.Codec == "SSA" {
						fileExt = "ass"
					} else if t.Codec == "vobsub" || t.Codec == "VobSub" {
						fileExt = "idx"
					} else {
						// Use lowercase codec name as fallback but remove any slashes
						cleanCodec := strings.ReplaceAll(t.Codec, "/", "_")
						fileExt = strings.ToLower(cleanCodec)
					}

					// Debug output for file naming
					fyne.Do(func() {
						result.SetText(result.Text + "\n\n=== Track Extraction ===\n")
						result.SetText(result.Text + fmt.Sprintf("Track: %d (%s - %s)\n", t.Num, t.Lang, t.Codec))
					})

					outFile = fmt.Sprintf("%s.track%d_%s.%s", mkvBaseName, t.Num, t.Lang, fileExt)

					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("Output file: %s\n", outFile))
					})
					// Use absolute paths for all subtitle extractions to avoid directory creation issues
					absOutFile := filepath.Join(outDir, outFile)
					cmd := exec.Command("mkvextract", "tracks", mkvPath, fmt.Sprintf("%d:%s", t.Num, absOutFile))

					fyne.Do(func() {
						result.SetText(result.Text + fmt.Sprintf("\nExtracting to: %s", absOutFile))
					})

					output, err = cmd.CombinedOutput()

					// Set proper file permissions for subtitle files (read/write for user, read for group/others)
					if err == nil {
						outFilePath := filepath.Join(outDir, outFile)
						os.Chmod(outFilePath, 0644) // rw-r--r--
					}
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

	// Create Support button with improved UX
	supportBtn := widget.NewButton("Donate ☕", func() {
		// Show a confirmation dialog with information about the donation
		confirm := dialog.NewConfirm(
			"Support Subtitle Forge",
			"Your donation helps maintain and improve Subtitle Forge. Would you like to proceed to PayPal?",
			func(ok bool) {
				if ok {
					supportURL, _ := url.Parse("https://paypal.me/VenimK")
					fyne.CurrentApp().OpenURL(supportURL)
				}
			},
			w,
		)
		confirm.SetDismissText("Cancel")
		confirm.SetConfirmText("Donate")
		confirm.Show()
	})
	supportBtn.Importance = widget.HighImportance

	// Create button row for better layout
	buttonRow := container.NewHBox(loadTracksBtn, startExtractBtn, layout.NewSpacer(), supportBtn)

	// Use app.NewWithID for better performance and to avoid preferences API warnings
	// This was already set at the beginning of main()

	// Use a more efficient layout with container.NewBorder for better performance
	// Create app title with version
	titleLabel := widget.NewLabel("Subtitle Forge v1.4")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	topContent := container.NewVBox(
		titleLabel,
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
		trackListScroll,
	)

	bottomContent := container.NewVBox(
		widget.NewLabel("Results:"),
		resultScroll,
		dependencyButtons,
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
