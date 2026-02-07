// src/index.ts
import { execSync, exec } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import * as https from 'https';

// --- CONFIGURATION ---
const CHECK_INTERVAL_MS = 10 * 60 * 1000; // 10 Minutes
const API_KEY = process.env.AI_API_KEY; // Ensure this is set in your environment
// Example using Google Gemini API URL (or swap for OpenAI)
const AI_ENDPOINT = `https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=${API_KEY}`;

// --- GIT HELPER FUNCTIONS ---

function runCommand(command: string): string {
    try {
        return execSync(command, { encoding: 'utf-8', stdio: 'pipe' }).trim();
    } catch (error: any) {
        return '';
    }
}

function hasChanges(): boolean {
    // Check for modified, added, or deleted files
    const status = runCommand('git status --porcelain');
    return status.length > 0;
}

function getDiff(): string {
    // Get diff of tracked files and list of untracked files
    const diff = runCommand('git diff');
    const untracked = runCommand('git ls-files --others --exclude-standard');
    return `Diff:\n${diff}\n\nUntracked Files:\n${untracked}`;
}

// --- AI GENERATION ---

async function generateCommitMessage(diff: string): Promise<string> {
    if (!API_KEY) return "Auto-commit: Work in progress (AI Key missing)";

    const prompt = `
        You are a git commit bot. 
        Analyze the following git diff and generate a concise, conventional commit message (e.g., "feat: ...", "fix: ...").
        Do not include quotes or extra text. Just the message.
        
        ${diff.substring(0, 3000)} 
        // Truncated to prevent token limits if diff is huge
    `;

    const payload = JSON.stringify({
        contents: [{ parts: [{ text: prompt }] }]
    });

    return new Promise((resolve, reject) => {
        const req = https.request(AI_ENDPOINT, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        }, (res) => {
            let data = '';
            res.on('data', (chunk) => data += chunk);
            res.on('end', () => {
                try {
                    const response = JSON.parse(data);
                    // Adjust parsing logic based on your specific AI provider's response structure
                    const message = response?.candidates?.[0]?.content?.parts?.[0]?.text || "Auto-commit: Updates";
                    resolve(message.trim());
                } catch (e) {
                    resolve("Auto-commit: Updates (AI Error)");
                }
            });
        });

        req.on('error', (e) => resolve("Auto-commit: Updates (Network Error)"));
        req.write(payload);
        req.end();
    });
}

// --- MAIN LOOP ---

async function main() {
    console.log(`[${new Date().toLocaleTimeString()}] Git Autobot started. Checking every 10 minutes...`);

    // Infinite loop to keep the process alive
    while (true) {
        try {
            if (hasChanges()) {
                console.log(`[${new Date().toLocaleTimeString()}] Changes detected.`);
                
                const diff = getDiff();
                console.log("Generating commit message...");
                
                const commitMsg = await generateCommitMessage(diff);
                console.log(`Commit Message: "${commitMsg}"`);

                runCommand('git add .');
                runCommand(`git commit -m "${commitMsg}"`);
                
                // Try pushing
                try {
                    console.log("Pushing to remote...");
                    execSync('git push', { stdio: 'inherit' });
                    console.log("Success!");
                } catch (err) {
                    console.error("Push failed. Check internet or credentials.");
                }
            } else {
                console.log(`[${new Date().toLocaleTimeString()}] No changes.`);
            }
        } catch (error) {
            console.error("Unexpected error in loop:", error);
        }

        // Wait for 10 minutes
        await new Promise(resolve => setTimeout(resolve, CHECK_INTERVAL_MS));
    }
}

main();
