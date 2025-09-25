original project structure:
# Battle Chess RPG - Complete Architecture Documentation

A JRPG-style chess game with elemental magic, player progression, and overworld exploration. Built with Go backend, JavaScript frontend, and SQLite persistence.

## Project Structure

```
battle_chess/
├── go.mod, go.sum                 # Go module dependencies
├── main.go                        # HTTP server & route handlers
├── start.bat, start.sh            # Platform launch scripts
├── battle_chess.exe               # Compiled Windows binary
├── 
├── auth/                          # Authentication & session management
│   └── auth.go                    # User registration, login, JWT sessions
├── 
├── engine/                        # Chess game engine
│   ├── engine.go                  # Board state, move validation, FEN parsing
│   ├── piece_attributes/          # Modular piece abilities system
│   │   ├── registry.go            # Central ability/augmentor lookup
│   │   ├── augmentMovement.go     # Movement-altering abilities
│   │   ├── augmentElement.go      # Elemental interaction effects
│   │   ├── augmentOffense.go      # Combat/capture modifiers
│   │   ├── augmentTemporal.go     # Time-based effects
│   │   ├── augmentConsumable.go   # One-time use abilities
│   │   └── helpers.go             # Utility functions for abilities
│   └── magicItem/                 # Equipment system
│       ├── internal/registry.go   # Item type registry
│       ├── blessings/             # Permanent stat bonuses
│       ├── controls/              # Tactical control items
│       └── gambles/               # High-risk/reward items
├── 
├── pieces/                        # Chess piece implementations
│   ├── piece.go                   # Base piece interface & movement system
│   ├── king.go                    # King piece with castling
│   ├── queen.go                   # Queen piece (sliding movement)
│   ├── rook.go                    # Rook piece (orthogonal sliding)
│   ├── bishop.go                  # Bishop piece (diagonal sliding)
│   ├── knight.go                  # Knight piece (L-shaped leaps)
│   └── pawn.go                    # Pawn piece (forward movement, en passant)
├── 
├── player/                        # Player progression system
│   ├── player.go                  # Player stats, leveling, equipment
│   ├── character.go               # Avatar customization
│   └── inventory.go               # Item management
├── 
├── storage/                       # Database persistence
│   └── storage.go                 # SQLite schema & CRUD operations
├── 
├── overworld/                     # RPG exploration layer
│   ├── map.go                     # Tile-based world maps
│   ├── npcs.go                    # AI trainers & interactions
│   └── renderer.go                # Map rendering for frontend
├── 
├── social/                        # Multiplayer features
│   ├── friends.go                 # Friend system & profiles
│   ├── chat.go                    # Real-time messaging
│   ├── lobby.go                   # Game matchmaking
│   ├── tournaments.go             # Competitive events
│   ├── achievements.go            # Progress tracking
│   ├── replays.go                 # Game recording/playback
│   └── webrtc.go                  # Voice/video calls
├── 
├── fields/                        # Battlefield environments
│   └── elementalFields.go         # Environmental effects on gameplay
├── 
├── web/                           # Frontend assets
│   ├── static/
│   │   ├── app.css                # Gothic Castlevania-inspired UI theme
│   │   ├── app.js                 # Chess board interaction & game logic
│   │   ├── overworld.js           # Map exploration interface
│   │   ├── social.js              # Chat, friends, lobby management
│   │   ├── webrtc.js              # Voice/video communication
│   │   ├── sprites/               # Piece artwork (Hellenic, Babylonian, Sikh)
│   │   ├── audio/                 # Sound effects & music
│   │   ├── maps/                  # JSON map definitions
│   │   └── characters/            # Avatar sprites
│   └── templates/                 # Go HTML templates
│       ├── layout.tmpl            # Base page structure
│       ├── landing.tmpl           # Marketing homepage
│       ├── login.tmpl, register.tmpl # Authentication forms
│       ├── index.tmpl             # Game lobby
│       ├── game.tmpl              # Chess game interface
│       ├── overworld.tmpl         # Map exploration
│       ├── profile.tmpl           # Player statistics
│       ├── settings.tmpl          # User preferences
│       └── admin.tmpl             # Administrative panel
└── 
└── worldMap/                      # Static map data files
``


Refactor instructions:
1. Executive Summary & Core Objectives
This document presents a unified and authoritative design for the "Battle Chess" project. It integrates the v1.0 refactor with the v1.1 and v1.2 modular extensions into a single, cohesive specification. The primary goal is to refactor the existing system for high performance and scalability while expanding its feature set, ensuring no loss of information from prior designs.
The core objectives are to:
Preserve Public APIs: Maintain facade stability for all public-facing surfaces while completely overhauling the internal architecture.
Modernize Internals: Transition the chess engine to a bitboard representation and implement ID/bitmask-based fast paths for core systems, moving away from string-based lookups in performance-critical code.
Enhance Network Transport: Replace legacy SSE/WebRTC components with a high-performance, binary WebSocket (RFC6455) transport system featuring backpressure management.
Expand Game World: Introduce new features including a sharded overworld with an Area of Interest (AOI) system, factions and political dynamics, player achievements, and an integrated chat system.
Standardize Game Entities: Unify abilities, augmentors, items, consumables, and temporal effects under a consistent system of numeric ID registries and bitmasks.
Adhere to Strict Technical Guardrails: Mandate a Go (1.22+) stdlib-only backend, a vanilla JavaScript frontend utilizing Go templates, and SQLite with WAL for persistence.
2. Technical Guardrails & Performance Requirements
This project adheres to a strict set of non-negotiable technical constraints to ensure performance, maintainability, and simplicity.
Backend: Go 1.22+ standard library is mandatory (e.g., net/http, database/sql, encoding/*, sync). No third-party frameworks are permitted.
Frontend: The client-side must be implemented with vanilla JavaScript and Go templates. The rendering will be done on HTML5 Canvas with dirty rectangle optimization, leveraging Web Workers and SharedArrayBuffer for performance. No JavaScript frameworks are to be used.
Persistence: All data will be stored in SQLite operating in Write-Ahead Logging (WAL) mode. Concurrency will be managed via optimistic concurrency control (OCC) on game state rows, and all database interactions must use prepared statements.
Transport Protocol: Communication between the authoritative server and clients will exclusively use RFC6455 WebSockets with binary opcodes. Server frames will be unmasked, and client frames must be masked. The protocol will support coalesced deltas for high-frequency updates and apply backpressure to manage network congestion.
Performance:
The server is authoritative, while the client is responsible for prediction and rendering only.
World shards must tick at a rate of 20 Hz, with politics updates occurring at 0.5–1 Hz.
There must be zero heap allocations in critical hot paths, including move generation, event emission, and augmentor application. This will be achieved through Structure of Arrays (SoA) data layouts, ring buffers, and bit packing.
A zero-copy write path for WebSocket encoding will be implemented using pooled buffers, with a 100ms deadline for reliable message sends.
3. System Architecture & Repository Layout
The project will be organized into a clean, modular structure to separate concerns and facilitate development. Remnants of the old SSE hub and WebRTC implementation will be removed.
Final Repository Structure:
code
Code
battle_chess/
├─ cmd/server/main.go
├─ internal/
│  ├─ httpx/       # HTTP server, middleware (security, auth), static files
│  ├─ ws/          # WebSocket connection management, ring buffers, codec
│  ├─ protocol/    # Binary protocol definitions, opcodes, encoding/decoding
│  ├─ world/       # Sharded overworld, SoA, AOI, streaming logic
│  ├─ faction/     # Faction politics, regional influence, treaties, events
│  ├─ game/        # Chess engine (bitboards, Zobrist hashing), FEN facade, ability logic
│  ├─ player/      # Player state, inventory, progression, effects
│  └─ persist/     # Database interaction layer, OCC, migrations
├─ web/
│  ├─ static/      # Vanilla JS (app.js, worker.js), CSS, and generated assets
│  └─ templates/   # Go templates for server-side rendering (layout, lobby, etc.)
├─ scripts/       # Utility scripts (e.g., magic table generation, seeding)
└─ sql/           # SQL schema, indices, and migration files
4. Networking & Communication
The standard library ServeMux will serve as the router, with middleware chained for recovery, logging, security, and authentication.
Endpoints:
GET: /, /login, /lobby, /game/{id}, /overworld, /me, /assets/*
POST: /login, /register
GET (Upgrade): /ws for WebSocket connections.
Security Headers: Strict security headers will be enforced:
COOP: same-origin
COEP: require-corp
CORP: same-origin
CSP: default-src 'self'; img-src 'self' data:; worker-src 'self'; connect-src 'self' wss:
HSTS: Enabled where HTTPS is used.
Session Management: Session cookies will be configured as Secure, HttpOnly, and SameSite=Lax.
The WebSocket layer is designed for high-frequency, low-latency communication with built-in flow control.
Buffering: Each connection uses a single-producer/single-consumer (SPSC) ring buffer to manage outgoing messages.
Message Classes: Messages are classified as either "delta" (can be coalesced) or "reliable" (must be sent). Reliable messages will block for a maximum of 100ms; if the buffer remains full, the client is disconnected.
Heartbeat: A PING is sent every 20 seconds. If a PONG is not received within 30 seconds, the connection is terminated.
Encoding: A pool of buffers is used for binary encoding of opcodes and data payloads to eliminate allocations in the hot path.
Opcodes (u8):
0x01: Hello (Server-to-Client)
0x02: Enter (Client-to-Server)
0x03: Delta (Server-to-Client)
0x04: Chat (Bidirectional)
0x05: Invite (Client-to-Server)
0x06: Battle (Server-to-Client)
0x07: Move (Client-to-Server)
0x08: Ack (Server-to-Client)
0x09: FacUp (Faction Update, Server-to-Client)
5. World & Overworld Systems
The game world is divided into shards, each running an independent 20 Hz simulation loop.
Loop Logic: Each tick, the shard drains client intents, integrates movement and collisions, marks dirty entities, and builds an OpDelta message for broadcast.
Area of Interest (AOI): A bucket grid system (32x32 tiles) manages entity visibility. Each subscriber receives updates for a maximum of 128 visible entities, sorted by Manhattan distance.
Handoff: Player movement across shard borders is handled via message-based intent posting to the neighboring shard's router.
To optimize for CPU cache performance, world entities are stored in a Structure of Arrays (SoA) layout.
ID[]: Entity identifiers.
X[], Y[]: Position coordinates.
VX[], VY[]: Velocity vectors.
Flags[]: Bitmask for entity state (e.g., PC, NPC, stealth, dirty).
The server computes viewport-specific dirty rectangles and streams OpDelta messages to the client. A JSON-based renderer is retained for debugging purposes only.
6. Chess Engine Refactor
The core chess engine will be rewritten to use bitboards for performance while maintaining its existing public API for backward compatibility.
Stable Facade: The existing Board API, including methods like NewStart, FromFEN, ToFEN, LegalMovesFrom, and Apply, will remain unchanged.
Internal Representation: The engine's internal state will be managed using bitboards for piece sets, occupancy, a Zobrist hash for transposition tables, and state variables for castling and en passant. Make/Unmake functions will use undo records for efficient state traversal.
Payload Packing: The game board state will be packed into a 256-byte format for efficient transmission in OpBattle WebSocket messages.
Performance Hints: To avoid dynamic slice growth during move generation, capacity hints will be pre-allocated for piece movements (e.g., King=8, Knight=8, Rook=14, Pawn=6).
Compatibility: Legacy coordinate arrays will be retained as compatibility shadows, used only for FEN generation and rendering paths.
7. Unified Entity & Effects System
Abilities, items, and other game modifiers are standardized under a unified, ID-driven system to eliminate string comparisons from hot paths.
ID Registries: All abilities (elemental, offensive, temporal, consumable) and movement augmentors are assigned unique numeric IDs (AbilityID, AugmentorID, etc.) and grouped by kind into bitmasks (AbilityMask).
ID-First Precedence: The system always prefers numeric IDs for logic processing. Name-based registries and key-based resolvers are kept as thin adapter layers for development, debugging, and legacy compatibility, but they are not used in performance-critical code.
Composition: A BasePiece struct composes entity attributes, holding an AbilityID, an AugMask, and collision flags. The ComposeModifiers function uses a fast path based on these IDs and a lookup table to apply effects.
Blessings, controls, and gambles are consolidated into a single "Effects" system driven by an EffectID with corresponding apply/remove function tables. Legacy registration functions will forward to this unified system.
Overworld regions can have "Elemental Fields" that apply minor biases. On region entry, the MapManager pre-compiles the field's effects into an AbilityMask, which is then applied to the player's army. This system influences minor modifiers but does not break core chess balance.
8. Gameplay Features
A political simulation layer adds strategic depth to the overworld.
Data Model: SQLite tables will store data for regions, factions, influence scores, treaties, and political events.
Tick Rate: The politics system ticks at a slower cadence of 0.5–1 Hz.
Updates: Influence deltas are broadcast to clients via the OpFacUp WebSocket message.
Achievements: A database-backed AchievementsManager listens to a small, internal event bus for game and chat events, unlocking achievements as criteria are met. The database will include indices on user achievements for fast lookups.
Chat: The ChatManager is also database-backed and is bridged to the WebSocket transport. It reuses the per-connection ring buffer policy to handle OpChat messages bidirectionally and enforce backpressure.
9. Persistence Layer
Schema: The existing schemas for games, moves, and authentication are retained. New tables are added for user_style (culture and grid preferences), user_achievements, and all faction/politics data.
Concurrency: The established Optimistic Concurrency Control (OCC) pattern will be used for all state-modifying database transactions.
Data Integrity: Foreign keys will be used where applicable to maintain relational integrity, particularly for user and achievement data.
10. Execution & Acceptance Plan
The refactor and expansion will be executed surgically to ensure a smooth transition.
Execution Order: The plan prioritizes foundational changes first, starting with the transport and protocol layers, followed by the world simulation, chess engine, ability systems, and finally frontend and feature integration.
Acceptance Criteria:
Functional: All legacy public APIs must compile and function correctly. Gameplay logic should be unchanged, aside from performance improvements and new features.
Transport: The WebSocket transport must guarantee zero reliable message drops and maintain ≤5% coalesced delta loss under load. Heartbeats must be strictly enforced.
Performance: Move generation and event emission hot paths must be allocation-free. AOI queries must complete within a p95 of 2ms. The frontend render loop must maintain its frame budget.
Compatibility: ID-backed fast paths must be verified through testing, while name-based registries remain functional off the hot paths