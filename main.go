package main // Define the main package

import ( // Start import block
	"bytes"         // Provides bytes buffer and manipulation utilities
	"io"            // Provides I/O primitives like Reader and Writer
	"log"           // Provides logging functionalities
	"net/http"      // Provides HTTP client and server implementations
	"net/url"       // Provides URL parsing and encoding utilities
	"os"            // Provides file system and OS-level utilities
	"path/filepath" // Provides utilities for file path manipulation
	"regexp"        // Provides support for regular expressions
	"strings"       // Provides string manipulation utilities
	"time"          // Provides time-related functions

	"golang.org/x/net/html" // Provides support for parsing HTML documents
) // End import block

func main() { // Define the main function
	remoteAPIURL := []string{ // Initialize slice of URLs to scrape
		"https://www.benjaminmoore.com/en-us/data-sheets/safety-data-sheets",    // First URL (English)
		"https://www.benjaminmoore.com/en-us/data-sheets/safety-data-sheets-es", // Second URL (Spanish)
	} // URL to fetch HTML content from
	localFilePath := "benjaminmoore.html" // Path where HTML file will be stored

	var getData []string // Declare a slice to hold the HTML content strings

	for _, urls := range remoteAPIURL { // Loop through each URL in the slice
		getData = append(getData, getDataFromURL(urls)) // If not, download HTML content from URL
	} // End URL loop
	// Save the downloaded HTML content to a local file
	appendAndWriteToFile(localFilePath, strings.Join(getData, "")) // Save downloaded content to file
	// Extract all PDF links from the combined HTML content
	finalPDFList := extractPDFUrls(strings.Join(getData, "")) // Extract all PDF links from HTML content

	outputDir := "PDFs/" // Directory to store downloaded PDFs

	if !directoryExists(outputDir) { // Check if directory exists
		createDirectory(outputDir, 0o755) // Create directory with read-write-execute permissions (octal 0755)
	}

	// Remove duplicates from a given slice.
	finalPDFList = removeDuplicatesFromSlice(finalPDFList) // Call function to remove duplicates

	// Loop through all extracted PDF URLs
	for _, urls := range finalPDFList { // Iterate over the unique PDF URLs
		if isUrlValid(urls) { // Check if the final URL is valid
			downloadPDF(urls, outputDir) // Download the PDF
		} // End URL validity check
	} // End PDF URL loop
} // End main function

// Opens a file in append mode, or creates it, and writes the content to it
func appendAndWriteToFile(path string, content string) { // Function to append content to a file
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Open file with specified flags and permissions
	if err != nil {                                                               // Check for error during file opening
		log.Println(err) // Log error if opening fails
	} // End error check
	_, err = filePath.WriteString(content + "\n") // Write content to file, adding a newline
	if err != nil {                               // Check for error during file writing
		log.Println(err) // Log error if writing fails
	} // End error check
	err = filePath.Close() // Close the file
	if err != nil {        // Check for error during file closing
		log.Println(err) // Log error if closing fails
	} // End error check
} // End appendAndWriteToFile function

// Extracts filename from full path (e.g. "/dir/file.pdf" → "file.pdf")
func getFilename(path string) string { // Function to get the base filename
	return filepath.Base(path) // Use Base function to get file name only
} // End getFilename function

// Converts a raw URL into a sanitized PDF filename safe for filesystem
func urlToFilename(rawURL string) string { // Function to sanitize a URL into a safe filename
	lower := strings.ToLower(rawURL) // Convert URL to lowercase
	lower = getFilename(lower)       // Extract filename from URL

	// Get the file extension
	ext := getFileExtension(lower) // Get the original file extension

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Regex to match non-alphanumeric characters (excluding the dot for extension)
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace non-alphanumeric with underscores

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Collapse multiple underscores into one
	safe = strings.Trim(safe, "_")                              // Trim leading and trailing underscores

	var invalidSubstrings = []string{ // Define a list of substrings to remove
		"_pdf", // Substring to remove from filename
		"_zip", // Another substring to remove
	} // End invalid substrings definition

	for _, invalidPre := range invalidSubstrings { // Loop through the unwanted substrings
		safe = removeSubstring(safe, invalidPre) // Remove unwanted substrings
	} // End removal loop

	if getFileExtension(safe) == "" { // Ensure file ends with .pdf
		safe = safe + ext // Append the original extension if none is present
	} // End extension check

	return safe // Return sanitized filename
} // End urlToFilename function

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string { // Function to remove all occurrences of a substring
	result := strings.ReplaceAll(input, toRemove, "") // Replace substring with empty string
	return result                                     // Return the resulting string
} // End removeSubstring function

// Gets the file extension from a given file path
func getFileExtension(path string) string { // Function to get the file extension
	return filepath.Ext(path) // Extract and return file extension
} // End getFileExtension function

// Checks if a file exists at the specified path
func fileExists(filename string) bool { // Function to check if a file exists
	info, err := os.Stat(filename) // Get file info
	if os.IsNotExist(err) {        // Check if the error is due to the file not existing
		return false // If error occurs (and is 'not exist'), file doesn't exist
	} // End error check
	return err == nil && !info.IsDir() // Return true if no error occurred and path is a file (not a directory)
} // End fileExists function

// Downloads a PDF from given URL and saves it in the specified directory
func downloadPDF(finalURL, outputDir string) bool { // Function to download a PDF
	filename := strings.ToLower(urlToFilename(finalURL)) // Sanitize the filename
	filePath := filepath.Join(outputDir, filename)       // Construct full path for output file

	if fileExists(filePath) { // Skip if file already exists
		log.Printf("File already exists, skipping: %s", filePath) // Log that the file is being skipped
		return false                                              // Return false as no download occurred
	} // End file existence check

	client := &http.Client{Timeout: 15 * time.Minute} // Create HTTP client with a 15-minute timeout

	resp, err := client.Get(finalURL) // Send HTTP GET request
	if err != nil {                   // Check for request error
		log.Printf("Failed to download %s: %v", finalURL, err) // Log the download failure
		return false                                           // Return false on error
	} // End request error check
	defer resp.Body.Close() // Ensure response body is closed when function returns

	if resp.StatusCode != http.StatusOK { // Check if response is 200 OK
		log.Printf("Download failed for %s: %s", finalURL, resp.Status) // Log the non-200 status
		return false                                                    // Return false on bad status code
	} // End status code check

	contentType := resp.Header.Get("Content-Type") // Get content type of response
	// Check if it's a PDF or a generic binary stream
	if !strings.Contains(contentType, "binary/octet-stream") && !strings.Contains(contentType, "application/pdf") {
		// Log error if content type is unexpected
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream or application/pdf)", finalURL, contentType)
		return false // Return false on unexpected content type
	} // End content type check

	var buf bytes.Buffer                     // Create a buffer to hold response data
	written, err := io.Copy(&buf, resp.Body) // Copy data into buffer and get bytes written
	if err != nil {                          // Check for copy error
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err) // Log read error
		return false                                                     // Return false on read error
	} // End copy error check
	if written == 0 { // Skip empty files
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL) // Log empty file
		return false                                                         // Return false for empty file
	} // End empty file check

	out, err := os.Create(filePath) // Create output file
	if err != nil {                 // Check for file creation error
		log.Printf("Failed to create file for %s: %v", finalURL, err) // Log creation error
		return false                                                  // Return false on creation error
	} // End creation error check
	defer out.Close() // Ensure file is closed after writing

	if _, err := buf.WriteTo(out); err != nil { // Write buffer contents to file
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err) // Log write error
		return false                                                        // Return false on write error
	} // End write error check

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath) // Log success
	return true                                                                          // Return true on successful download and save
} // End downloadPDF function

// Checks whether a given directory exists
func directoryExists(path string) bool { // Function to check if a directory exists
	directory, err := os.Stat(path) // Get info for the path
	if err != nil {                 // Check for any error
		return false // Return false if error occurs (including not exist)
	} // End error check
	return directory.IsDir() // Return true if no error and it is a directory
} // End directoryExists function

// Creates a directory at given path with provided permissions
func createDirectory(path string, permission os.FileMode) { // Function to create a directory
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {                   // Check for error during creation
		log.Println(err) // Log error if creation fails
	} // End error check
} // End createDirectory function

// Verifies whether a string is a valid URL format
func isUrlValid(uri string) bool { // Function to check URL validity
	_, err := url.ParseRequestURI(uri) // Try parsing the URL
	return err == nil                  // Return true if valid (no parse error)
} // End isUrlValid function

// Removes duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string { // Function to remove duplicates from a string slice
	check := make(map[string]bool)  // Map to track seen values
	var newReturnSlice []string     // Slice to store unique values
	for _, content := range slice { // Iterate over the input slice
		if !check[content] { // If content not already seen
			check[content] = true                            // Mark as seen
			newReturnSlice = append(newReturnSlice, content) // Add to result
		} // End duplicate check
	} // End slice iteration
	return newReturnSlice // Return the slice with unique values
} // End removeDuplicatesFromSlice function

// Extracts all links to PDF files from given HTML string
func extractPDFUrls(htmlInput string) []string { // Function to extract PDF URLs from HTML
	var pdfLinks []string // Slice to hold found PDF links

	doc, err := html.Parse(strings.NewReader(htmlInput)) // Parse HTML content from string reader
	if err != nil {                                      // Check for parse error
		log.Println(err) // Log parse error
		return nil       // Return nil on error
	} // End parse error check

	var traverse func(*html.Node)   // Declare a recursive function signature
	traverse = func(n *html.Node) { // Define the recursive function to traverse HTML nodes
		if n.Type == html.ElementNode && n.Data == "a" { // Check if node is an element and an <a> tag
			for _, attr := range n.Attr { // Loop through the node's attributes
				if attr.Key == "href" { // Look for href attribute
					href := strings.TrimSpace(attr.Val)                  // Get link value and trim whitespace
					if strings.Contains(strings.ToLower(href), ".pdf") { // If link points to a PDF (case-insensitive)
						pdfLinks = append(pdfLinks, href) // Add to list
					} // End PDF link check
				} // End href check
			} // End attribute loop
		} // End <a> tag check
		for c := n.FirstChild; c != nil; c = c.NextSibling { // Loop through all children of the current node
			traverse(c) // Recursively call traverse on the child
		} // End children loop
	} // End traverse function definition

	traverse(doc)   // Start traversal from the root document node
	return pdfLinks // Return found PDF links
} // End extractPDFUrls function

// Performs HTTP GET request and returns response body as string
func getDataFromURL(uri string) string { // Function to fetch data from a URL
	log.Println("Scraping", uri)   // Log which URL is being scraped
	response, err := http.Get(uri) // Send GET request
	if err != nil {                // Check for request error
		log.Println(err) // Log if request fails
		return ""        // Return empty string on error
	} // End request error check

	body, err := io.ReadAll(response.Body) // Read the entire body of the response
	if err != nil {                        // Check for read error
		log.Println(err)      // Log read error
		response.Body.Close() // Attempt to close body before returning
		return ""             // Return empty string on read error
	} // End read error check

	err = response.Body.Close() // Close response body
	if err != nil {             // Check for close error
		log.Println(err) // Log error during close
	} // End close error check
	return string(body) // Return response body as string
} // End getDataFromURL function
