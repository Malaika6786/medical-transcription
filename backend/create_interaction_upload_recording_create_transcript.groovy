// Corti API - Create Interaction and Upload Recording
// Run with: node corti-upload.js

const fs = require('fs');
const path = require('path');

const CONFIG = {
  CLIENT_ID: "YOUR API CLIENT",
  CLIENT_SECRET: "YOUR CLIENT SECRET",
  ENVIRONMENT: "eu",
  TENANT: "base",
  AUDIO_FILE_PATH: "YOUR FILE PATH"
};

async function getAccessToken() {
  const tokenUrl = `https://auth.${CONFIG.ENVIRONMENT}.corti.app/realms/${CONFIG.TENANT}/protocol/openid-connect/token`;
  
  const params = new URLSearchParams();
  params.append("client_id", CONFIG.CLIENT_ID);
  params.append("client_secret", CONFIG.CLIENT_SECRET);
  params.append("grant_type", "client_credentials");
  params.append("scope", "openid");

  console.log("Authenticating with Corti API...");
  
  const response = await fetch(tokenUrl, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Authentication failed (${response.status}): ${errorText}`);
  }

  const data = await response.json();
  console.log("Authentication successful!");
  return data.access_token;
}

async function createInteraction(accessToken) {
  const apiUrl = `https://api.${CONFIG.ENVIRONMENT}.corti.app/v2/interactions/`;
  
  const interactionData = {
    encounter: {
      identifier: `encounter-${Date.now()}`,
      status: "in-progress",
      type: "consultation",
      period: {
        startedAt: new Date().toISOString()
      },
      title: "Audio Upload - audio-sample-1.mp3"
    },
    patient: {
      identifier: `patient-${Date.now()}`,
      name: "Test Patient"
    }
  };

  console.log("\nCreating interaction...");
  
  const response = await fetch(apiUrl, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "Content-Type": "application/json",
      "Tenant-Name": CONFIG.TENANT
    },
    body: JSON.stringify(interactionData)
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Failed to create interaction (${response.status}): ${errorText}`);
  }

  const result = await response.json();
  console.log(`Interaction created successfully!`);
  console.log(`   Interaction ID: ${result.interactionId}`);
  return result.interactionId;
}

async function uploadRecording(accessToken, interactionId) {
  const apiUrl = `https://api.${CONFIG.ENVIRONMENT}.corti.app/v2/interactions/${interactionId}/recordings/`;
  
  if (!fs.existsSync(CONFIG.AUDIO_FILE_PATH)) {
    throw new Error(`Audio file not found at: ${CONFIG.AUDIO_FILE_PATH}`);
  }

  const audioBuffer = fs.readFileSync(CONFIG.AUDIO_FILE_PATH);
  const fileSizeMB = (audioBuffer.length / (1024 * 1024)).toFixed(2);
  
  console.log(`\nUploading recording: ${path.basename(CONFIG.AUDIO_FILE_PATH)}`);
  console.log(`   File size: ${fileSizeMB} MB`);
  
  if (audioBuffer.length > 150 * 1024 * 1024) {
    throw new Error("File size exceeds 150MB limit");
  }

  const response = await fetch(apiUrl, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "Content-Type": "application/octet-stream",
      "Tenant-Name": CONFIG.TENANT
    },
    body: audioBuffer
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Failed to upload recording (${response.status}): ${errorText}`);
  }

  const result = await response.json();
  console.log(`Recording uploaded successfully!`);
  console.log(`   Recording ID: ${result.recordingId}`);
  return result.recordingId;
}

async function createTranscript(accessToken, interactionId, recordingId) {
  const apiUrl = `https://api.${CONFIG.ENVIRONMENT}.corti.app/v2/interactions/${interactionId}/transcripts/`;
  
  const transcriptData = {
    recordingId: recordingId,
    primaryLanguage: "en"
  };

  console.log(`\nCreating transcript from recording...`);
  
  const response = await fetch(apiUrl, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "Content-Type": "application/json",
      "Tenant-Name": CONFIG.TENANT
    },
    body: JSON.stringify(transcriptData)
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Failed to create transcript (${response.status}): ${errorText}`);
  }

  const result = await response.json();
  console.log(`Transcript created successfully!`);
  console.log(`   Transcript ID: ${result.id}`);
  return result.id;
}

async function main() {
  console.log("====================================================");
  console.log("  Corti API - Interaction & Recording Upload");
  console.log("====================================================\n");

  try {
    const accessToken = await getAccessToken();
    const interactionId = await createInteraction(accessToken);
    const recordingId = await uploadRecording(accessToken, interactionId);
    const transcriptId = await createTranscript(accessToken, interactionId, recordingId);
    
    console.log("\n====================================================");
    console.log("              SUCCESS SUMMARY");
    console.log("====================================================");
    console.log(`Interaction ID: ${interactionId}`);
    console.log(`Recording ID:   ${recordingId}`);
    console.log(`Transcript ID:  ${transcriptId}`);
    console.log("\nAll operations completed successfully!");
    
  } catch (error) {
    console.error("\nERROR:", error.message);
    process.exit(1);
  }
}

main();