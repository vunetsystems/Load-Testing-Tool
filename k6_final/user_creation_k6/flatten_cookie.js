import { SharedArray } from 'k6/data';
import papaparse from 'https://jslib.k6.io/papaparse/5.1.1/index.js';
import { write } from 'k6/data';

// Function to flatten a nested object
function flattenObject(obj, prefix = '') {
  return Object.keys(obj).reduce((acc, k) => {
    const pre = prefix.length ? `${prefix}.` : '';
    if (typeof obj[k] === 'object' && obj[k] !== null && !Array.isArray(obj[k])) {
      Object.assign(acc, flattenObject(obj[k], `${pre}${k}`));
    } else {
      acc[`${pre}${k}`] = obj[k];
    }
    return acc;
  }, {});
}

// A more direct approach to parse cookies based on the exact format from the example
function parseCookies(cookieStr) {
  // Start with an empty object
  const cookies = {};
  
  // Remove the outer braces
  const trimmedStr = cookieStr.trim().slice(1, -1);
  
  // For debugging
  console.log("Processing cookie string:", trimmedStr.slice(0, 100) + "...");
  
  // Split by commas, but not inside quotes
  let inQuotes = false;
  let currentToken = "";
  const tokens = [];
  
  for (let i = 0; i < trimmedStr.length; i++) {
    const char = trimmedStr[i];
    
    if (char === "'" && (i === 0 || trimmedStr[i-1] !== '\\')) {
      inQuotes = !inQuotes;
    }
    
    if (char === ',' && !inQuotes) {
      tokens.push(currentToken.trim());
      currentToken = "";
    } else {
      currentToken += char;
    }
  }
  
  // Don't forget the last token
  if (currentToken) {
    tokens.push(currentToken.trim());
  }
  
  // Process each token as a key-value pair
  tokens.forEach(token => {
    const colonPos = token.indexOf(':');
    if (colonPos !== -1) {
      // Extract key (remove quotes)
      let key = token.substring(0, colonPos).trim();
      if (key.startsWith("'") && key.endsWith("'")) {
        key = key.slice(1, -1);
      }
      
      // Extract value (remove quotes if string)
      let value = token.substring(colonPos + 1).trim();
      if (value.startsWith("'") && value.endsWith("'")) {
        value = value.slice(1, -1);
      }
      
      cookies[key] = value;
    }
  });
  
  return cookies;
}

// Read the input CSV file with raw content
const csvData = new SharedArray('credentials', function() {
  // For debugging purposes, read raw file content first
  const rawFile = open('user_cookie.csv', 'r');
  console.log("Raw CSV first 200 chars:", rawFile.slice(0, 200));
  
  const parsed = papaparse.parse(rawFile, { 
    header: false,
    skipEmptyLines: true 
  });
  
  // Print info about the parsed data
  console.log(`CSV parsed: ${parsed.data.length} rows found`);
  if (parsed.data.length > 0) {
    console.log(`First row has ${parsed.data[0].length} columns`);
  }
  
  return parsed.data;
});

export default function() {
  const outputData = [];
  
  console.log(`Processing ${csvData.length} rows from CSV`);
  
  // Process each row
  for (let i = 0; i < csvData.length; i++) {
    const row = csvData[i];
    console.log(`Row ${i+1} raw data:`, JSON.stringify(row));
    
    if (row.length >= 3) {
      const username = row[0];
      const password = row[1];
      const cookieString = row[2];
      
      console.log(`Row ${i+1}: Username=${username}, Password=${password}`);
      console.log(`Cookie string from CSV: ${cookieString.substring(0, 50)}...`);
      
      try {
        // Use our direct parser
        const cookies = parseCookies(cookieString);
        
        // Check if we got something useful
        if (Object.keys(cookies).length > 0) {
          console.log(`Successfully parsed ${Object.keys(cookies).length} cookies`);
          
          // Flatten the cookie object
          const flattenedCookies = flattenObject(cookies);
	
	  console.log(flattenedCookies);
          
          // Create a new row with username, password, and flattened cookies
          const newRow = {
            username: username,
            password: password,
            ...flattenedCookies
          };
          
          outputData.push(newRow);
        } else {
          console.error(`No cookies parsed from row ${i+1}`);
        }
      } catch (e) {
        console.error(`Error processing row ${i+1}: ${e.message}`);
      }
    } else {
      console.error(`Row ${i+1} does not have enough columns (${row.length} found, expected 3)`);
    }
  }
  
  if (outputData.length === 0) {
    console.error("No data processed successfully. Check the input format.");
    return;
  }
  
  // Get all unique keys from the flattened data
  const allKeys = new Set();
  outputData.forEach(row => {
    Object.keys(row).forEach(key => {
      allKeys.add(key);
    });
  });
  
  // Create CSV header
  const headerRow = Array.from(allKeys);
  
  // Create CSV content
  let csvContent = headerRow.join(',') + '\n';
  
  outputData.forEach(row => {
    const csvRow = headerRow.map(key => {
      const value = row[key] !== undefined ? row[key] : '';
      // Escape commas and quotes
      const escapedValue = String(value).replace(/"/g, '""');
      return `"${escapedValue}"`;
    });
    csvContent += csvRow.join(',') + '\n';
  });
 
  console.log(csvContent)
  // Write output to file
  const outputFilePath = 'flattened_output.csv';
  write(csvContent, outputFilePath);
  
  console.log(`Flattened data written to ${outputFilePath}`);
}
