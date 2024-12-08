/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

var Guacamole = {};

/**
 * A reader which automatically handles the given input stream, returning
 * strictly received packets as array buffers. Note that this object will
 * overwrite any installed event handlers on the given Guacamole.InputStream.
 *
 * @constructor
 * @param {!Guacamole.InputStream} stream
 *     The stream that data will be read from.
 */
Guacamole.ArrayBufferReader = function (stream) {
    /**
     * Reference to this Guacamole.InputStream.
     * @private
     */
    var guac_reader = this;

    // Receive blobs as array buffers
    stream.onblob = function (data) {
        // Convert to ArrayBuffer
        var binary = window.atob(data);
        var arrayBuffer = new ArrayBuffer(binary.length);
        var bufferView = new Uint8Array(arrayBuffer);

        for (var i = 0; i < binary.length; i++) bufferView[i] = binary.charCodeAt(i);

        // Call handler, if present
        if (guac_reader.ondata) guac_reader.ondata(arrayBuffer);
    };

    // Simply call onend when end received
    stream.onend = function () {
        if (guac_reader.onend) guac_reader.onend();
    };

    /**
     * Fired once for every blob of data received.
     *
     * @event
     * @param {!ArrayBuffer} buffer
     *     The data packet received.
     */
    this.ondata = null;

    /**
     * Fired once this stream is finished and no further data will be written.
     * @event
     */
    this.onend = null;
};

/**
 * A writer which automatically writes to the given output stream with arbitrary
 * binary data, supplied as ArrayBuffers.
 *
 * @constructor
 * @param {!Guacamole.OutputStream} stream
 *     The stream that data will be written to.
 */
Guacamole.ArrayBufferWriter = function (stream) {
    /**
     * Reference to this Guacamole.StringWriter.
     *
     * @private
     * @type {!Guacamole.ArrayBufferWriter}
     */
    var guac_writer = this;

    // Simply call onack for acknowledgements
    stream.onack = function (status) {
        if (guac_writer.onack) guac_writer.onack(status);
    };

    /**
     * Encodes the given data as base64, sending it as a blob. The data must
     * be small enough to fit into a single blob instruction.
     *
     * @private
     * @param {!Uint8Array} bytes
     *     The data to send.
     */
    function __send_blob(bytes) {
        var binary = '';

        // Produce binary string from bytes in buffer
        for (var i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i]);

        // Send as base64
        stream.sendBlob(window.btoa(binary));
    }

    /**
     * The maximum length of any blob sent by this Guacamole.ArrayBufferWriter,
     * in bytes. Data sent via
     * [sendData()]{@link Guacamole.ArrayBufferWriter#sendData} which exceeds
     * this length will be split into multiple blobs. As the Guacamole protocol
     * limits the maximum size of any instruction or instruction element to
     * 8192 bytes, and the contents of blobs will be base64-encoded, this value
     * should only be increased with extreme caution.
     *
     * @type {!number}
     * @default {@link Guacamole.ArrayBufferWriter.DEFAULT_BLOB_LENGTH}
     */
    this.blobLength = Guacamole.ArrayBufferWriter.DEFAULT_BLOB_LENGTH;

    /**
     * Sends the given data.
     *
     * @param {!(ArrayBuffer|TypedArray)} data
     *     The data to send.
     */
    this.sendData = function (data) {
        var bytes = new Uint8Array(data);

        // If small enough to fit into single instruction, send as-is
        if (bytes.length <= guac_writer.blobLength) __send_blob(bytes);
        // Otherwise, send as multiple instructions
        else {
            for (var offset = 0; offset < bytes.length; offset += guac_writer.blobLength) __send_blob(bytes.subarray(offset, offset + guac_writer.blobLength));
        }
    };

    /**
     * Signals that no further text will be sent, effectively closing the
     * stream.
     */
    this.sendEnd = function () {
        stream.sendEnd();
    };

    /**
     * Fired for received data, if acknowledged by the server.
     * @event
     * @param {!Guacamole.Status} status
     *     The status of the operation.
     */
    this.onack = null;
};

/**
 * The default maximum blob length for new Guacamole.ArrayBufferWriter
 * instances.
 *
 * @constant
 * @type {!number}
 */
Guacamole.ArrayBufferWriter.DEFAULT_BLOB_LENGTH = 6048;

/**
 * Maintains a singleton instance of the Web Audio API AudioContext class,
 * instantiating the AudioContext only in response to the first call to
 * getAudioContext(), and only if no existing AudioContext instance has been
 * provided via the singleton property. Subsequent calls to getAudioContext()
 * will return the same instance.
 *
 * @namespace
 */
Guacamole.AudioContextFactory = {
    /**
     * A singleton instance of a Web Audio API AudioContext object, or null if
     * no instance has yes been created. This property may be manually set if
     * you wish to supply your own AudioContext instance, but care must be
     * taken to do so as early as possible. Assignments to this property will
     * not retroactively affect the value returned by previous calls to
     * getAudioContext().
     *
     * @type {AudioContext}
     */
    singleton: null,

    /**
     * Returns a singleton instance of a Web Audio API AudioContext object.
     *
     * @return {AudioContext}
     *     A singleton instance of a Web Audio API AudioContext object, or null
     *     if the Web Audio API is not supported.
     */
    getAudioContext: function getAudioContext() {
        // Fallback to Webkit-specific AudioContext implementation
        var AudioContext = window.AudioContext || window.webkitAudioContext;

        // Get new AudioContext instance if Web Audio API is supported
        if (AudioContext) {
            try {
                // Create new instance if none yet exists
                if (!Guacamole.AudioContextFactory.singleton) Guacamole.AudioContextFactory.singleton = new AudioContext();

                // Return singleton instance
                return Guacamole.AudioContextFactory.singleton;
            } catch (e) {
                // Do not use Web Audio API if not allowed by browser
            }
        }

        // Web Audio API not supported
        return null;
    },
};

/**
 * Abstract audio player which accepts, queues and plays back arbitrary audio
 * data. It is up to implementations of this class to provide some means of
 * handling a provided Guacamole.InputStream. Data received along the provided
 * stream is to be played back immediately.
 *
 * @constructor
 */
Guacamole.AudioPlayer = function AudioPlayer() {
    /**
     * Notifies this Guacamole.AudioPlayer that all audio up to the current
     * point in time has been given via the underlying stream, and that any
     * difference in time between queued audio data and the current time can be
     * considered latency.
     */
    this.sync = function sync() {
        // Default implementation - do nothing
    };
};

/**
 * Determines whether the given mimetype is supported by any built-in
 * implementation of Guacamole.AudioPlayer, and thus will be properly handled
 * by Guacamole.AudioPlayer.getInstance().
 *
 * @param {!string} mimetype
 *     The mimetype to check.
 *
 * @returns {!boolean}
 *     true if the given mimetype is supported by any built-in
 *     Guacamole.AudioPlayer, false otherwise.
 */
Guacamole.AudioPlayer.isSupportedType = function isSupportedType(mimetype) {
    return Guacamole.RawAudioPlayer.isSupportedType(mimetype);
};

/**
 * Returns a list of all mimetypes supported by any built-in
 * Guacamole.AudioPlayer, in rough order of priority. Beware that only the core
 * mimetypes themselves will be listed. Any mimetype parameters, even required
 * ones, will not be included in the list. For example, "audio/L8" is a
 * supported raw audio mimetype that is supported, but it is invalid without
 * additional parameters. Something like "audio/L8;rate=44100" would be valid,
 * however (see https://tools.ietf.org/html/rfc4856).
 *
 * @returns {!string[]}
 *     A list of all mimetypes supported by any built-in Guacamole.AudioPlayer,
 *     excluding any parameters.
 */
Guacamole.AudioPlayer.getSupportedTypes = function getSupportedTypes() {
    return Guacamole.RawAudioPlayer.getSupportedTypes();
};

/**
 * Returns an instance of Guacamole.AudioPlayer providing support for the given
 * audio format. If support for the given audio format is not available, null
 * is returned.
 *
 * @param {!Guacamole.InputStream} stream
 *     The Guacamole.InputStream to read audio data from.
 *
 * @param {!string} mimetype
 *     The mimetype of the audio data in the provided stream.
 *
 * @return {Guacamole.AudioPlayer}
 *     A Guacamole.AudioPlayer instance supporting the given mimetype and
 *     reading from the given stream, or null if support for the given mimetype
 *     is absent.
 */
Guacamole.AudioPlayer.getInstance = function getInstance(stream, mimetype) {
    // Use raw audio player if possible
    if (Guacamole.RawAudioPlayer.isSupportedType(mimetype)) return new Guacamole.RawAudioPlayer(stream, mimetype);

    // No support for given mimetype
    return null;
};

/**
 * Implementation of Guacamole.AudioPlayer providing support for raw PCM format
 * audio. This player relies only on the Web Audio API and does not require any
 * browser-level support for its audio formats.
 *
 * @constructor
 * @augments Guacamole.AudioPlayer
 * @param {!Guacamole.InputStream} stream
 *     The Guacamole.InputStream to read audio data from.
 *
 * @param {!string} mimetype
 *     The mimetype of the audio data in the provided stream, which must be a
 *     "audio/L8" or "audio/L16" mimetype with necessary parameters, such as:
 *     "audio/L16;rate=44100,channels=2".
 */
Guacamole.RawAudioPlayer = function RawAudioPlayer(stream, mimetype) {
    /**
     * The format of audio this player will decode.
     *
     * @private
     * @type {Guacamole.RawAudioFormat}
     */
    var format = Guacamole.RawAudioFormat.parse(mimetype);

    /**
     * An instance of a Web Audio API AudioContext object, or null if the
     * Web Audio API is not supported.
     *
     * @private
     * @type {AudioContext}
     */
    var context = Guacamole.AudioContextFactory.getAudioContext();

    /**
     * The earliest possible time that the next packet could play without
     * overlapping an already-playing packet, in seconds. Note that while this
     * value is in seconds, it is not an integer value and has microsecond
     * resolution.
     *
     * @private
     * @type {!number}
     */
    var nextPacketTime = context.currentTime;

    /**
     * Guacamole.ArrayBufferReader wrapped around the audio input stream
     * provided with this Guacamole.RawAudioPlayer was created.
     *
     * @private
     * @type {!Guacamole.ArrayBufferReader}
     */
    var reader = new Guacamole.ArrayBufferReader(stream);

    /**
     * The minimum size of an audio packet split by splitAudioPacket(), in
     * seconds. Audio packets smaller than this will not be split, nor will the
     * split result of a larger packet ever be smaller in size than this
     * minimum.
     *
     * @private
     * @constant
     * @type {!number}
     */
    var MIN_SPLIT_SIZE = 0.02;

    /**
     * The maximum amount of latency to allow between the buffered data stream
     * and the playback position, in seconds. Initially, this is set to
     * roughly one third of a second.
     *
     * @private
     * @type {!number}
     */
    var maxLatency = 0.3;

    /**
     * The type of typed array that will be used to represent each audio packet
     * internally. This will be either Int8Array or Int16Array, depending on
     * whether the raw audio format is 8-bit or 16-bit.
     *
     * @private
     * @constructor
     */
    var SampleArray = format.bytesPerSample === 1 ? window.Int8Array : window.Int16Array;

    /**
     * The maximum absolute value of any sample within a raw audio packet
     * received by this audio player. This depends only on the size of each
     * sample, and will be 128 for 8-bit audio and 32768 for 16-bit audio.
     *
     * @private
     * @type {!number}
     */
    var maxSampleValue = format.bytesPerSample === 1 ? 128 : 32768;

    /**
     * The queue of all pending audio packets, as an array of sample arrays.
     * Audio packets which are pending playback will be added to this queue for
     * further manipulation prior to scheduling via the Web Audio API. Once an
     * audio packet leaves this queue and is scheduled via the Web Audio API,
     * no further modifications can be made to that packet.
     *
     * @private
     * @type {!SampleArray[]}
     */
    var packetQueue = [];

    /**
     * Given an array of audio packets, returns a single audio packet
     * containing the concatenation of those packets.
     *
     * @private
     * @param {!SampleArray[]} packets
     *     The array of audio packets to concatenate.
     *
     * @returns {SampleArray}
     *     A single audio packet containing the concatenation of all given
     *     audio packets. If no packets are provided, this will be undefined.
     */
    var joinAudioPackets = function joinAudioPackets(packets) {
        // Do not bother joining if one or fewer packets are in the queue
        if (packets.length <= 1) return packets[0];

        // Determine total sample length of the entire queue
        var totalLength = 0;
        packets.forEach(function addPacketLengths(packet) {
            totalLength += packet.length;
        });

        // Append each packet within queue
        var offset = 0;
        var joined = new SampleArray(totalLength);
        packets.forEach(function appendPacket(packet) {
            joined.set(packet, offset);
            offset += packet.length;
        });

        return joined;
    };

    /**
     * Given a single packet of audio data, splits off an arbitrary length of
     * audio data from the beginning of that packet, returning the split result
     * as an array of two packets. The split location is determined through an
     * algorithm intended to minimize the liklihood of audible clicking between
     * packets. If no such split location is possible, an array containing only
     * the originally-provided audio packet is returned.
     *
     * @private
     * @param {!SampleArray} data
     *     The audio packet to split.
     *
     * @returns {!SampleArray[]}
     *     An array of audio packets containing the result of splitting the
     *     provided audio packet. If splitting is possible, this array will
     *     contain two packets. If splitting is not possible, this array will
     *     contain only the originally-provided packet.
     */
    var splitAudioPacket = function splitAudioPacket(data) {
        var minValue = Number.MAX_VALUE;
        var optimalSplitLength = data.length;

        // Calculate number of whole samples in the provided audio packet AND
        // in the minimum possible split packet
        var samples = Math.floor(data.length / format.channels);
        var minSplitSamples = Math.floor(format.rate * MIN_SPLIT_SIZE);

        // Calculate the beginning of the "end" of the audio packet
        var start = Math.max(format.channels * minSplitSamples, format.channels * (samples - minSplitSamples));

        // For all samples at the end of the given packet, find a point where
        // the perceptible volume across all channels is lowest (and thus is
        // the optimal point to split)
        for (var offset = start; offset < data.length; offset += format.channels) {
            // Calculate the sum of all values across all channels (the result
            // will be proportional to the average volume of a sample)
            var totalValue = 0;
            for (var channel = 0; channel < format.channels; channel++) {
                totalValue += Math.abs(data[offset + channel]);
            }

            // If this is the smallest average value thus far, set the split
            // length such that the first packet ends with the current sample
            if (totalValue <= minValue) {
                optimalSplitLength = offset + format.channels;
                minValue = totalValue;
            }
        }

        // If packet is not split, return the supplied packet untouched
        if (optimalSplitLength === data.length) return [data];

        // Otherwise, split the packet into two new packets according to the
        // calculated optimal split length
        return [
            new SampleArray(data.buffer.slice(0, optimalSplitLength * format.bytesPerSample)),
            new SampleArray(data.buffer.slice(optimalSplitLength * format.bytesPerSample)),
        ];
    };

    /**
     * Pushes the given packet of audio data onto the playback queue. Unlike
     * other private functions within Guacamole.RawAudioPlayer, the type of the
     * ArrayBuffer packet of audio data here need not be specific to the type
     * of audio (as with SampleArray). The ArrayBuffer type provided by a
     * Guacamole.ArrayBufferReader, for example, is sufficient. Any necessary
     * conversions will be performed automatically internally.
     *
     * @private
     * @param {!ArrayBuffer} data
     *     A raw packet of audio data that should be pushed onto the audio
     *     playback queue.
     */
    var pushAudioPacket = function pushAudioPacket(data) {
        packetQueue.push(new SampleArray(data));
    };

    /**
     * Shifts off and returns a packet of audio data from the beginning of the
     * playback queue. The length of this audio packet is determined
     * dynamically according to the click-reduction algorithm implemented by
     * splitAudioPacket().
     *
     * @private
     * @returns {SampleArray}
     *     A packet of audio data pulled from the beginning of the playback
     *     queue. If there is no audio currently in the playback queue, this
     *     will be null.
     */
    var shiftAudioPacket = function shiftAudioPacket() {
        // Flatten data in packet queue
        var data = joinAudioPackets(packetQueue);
        if (!data) return null;

        // Pull an appropriate amount of data from the front of the queue
        packetQueue = splitAudioPacket(data);
        data = packetQueue.shift();

        return data;
    };

    /**
     * Converts the given audio packet into an AudioBuffer, ready for playback
     * by the Web Audio API. Unlike the raw audio packets received by this
     * audio player, AudioBuffers require floating point samples and are split
     * into isolated planes of channel-specific data.
     *
     * @private
     * @param {!SampleArray} data
     *     The raw audio packet that should be converted into a Web Audio API
     *     AudioBuffer.
     *
     * @returns {!AudioBuffer}
     *     A new Web Audio API AudioBuffer containing the provided audio data,
     *     converted to the format used by the Web Audio API.
     */
    var toAudioBuffer = function toAudioBuffer(data) {
        // Calculate total number of samples
        var samples = data.length / format.channels;

        // Determine exactly when packet CAN play
        var packetTime = context.currentTime;
        if (nextPacketTime < packetTime) nextPacketTime = packetTime;

        // Get audio buffer for specified format
        var audioBuffer = context.createBuffer(format.channels, samples, format.rate);

        // Convert each channel
        for (var channel = 0; channel < format.channels; channel++) {
            var audioData = audioBuffer.getChannelData(channel);

            // Fill audio buffer with data for channel
            var offset = channel;
            for (var i = 0; i < samples; i++) {
                audioData[i] = data[offset] / maxSampleValue;
                offset += format.channels;
            }
        }

        return audioBuffer;
    };

    // Defer playback of received audio packets slightly
    reader.ondata = function playReceivedAudio(data) {
        // Push received samples onto queue
        pushAudioPacket(new SampleArray(data));

        // Shift off an arbitrary packet of audio data from the queue (this may
        // be different in size from the packet just pushed)
        var packet = shiftAudioPacket();
        if (!packet) return;

        // Determine exactly when packet CAN play
        var packetTime = context.currentTime;
        if (nextPacketTime < packetTime) nextPacketTime = packetTime;

        // Set up buffer source
        var source = context.createBufferSource();
        source.connect(context.destination);

        // Use noteOn() instead of start() if necessary
        if (!source.start) source.start = source.noteOn;

        // Schedule packet
        source.buffer = toAudioBuffer(packet);
        source.start(nextPacketTime);

        // Update timeline by duration of scheduled packet
        nextPacketTime += packet.length / format.channels / format.rate;
    };

    /** @override */
    this.sync = function sync() {
        // Calculate elapsed time since last sync
        var now = context.currentTime;

        // Reschedule future playback time such that playback latency is
        // bounded within a reasonable latency threshold
        nextPacketTime = Math.min(nextPacketTime, now + maxLatency);
    };
};

Guacamole.RawAudioPlayer.prototype = new Guacamole.AudioPlayer();

/**
 * Determines whether the given mimetype is supported by
 * Guacamole.RawAudioPlayer.
 *
 * @param {!string} mimetype
 *     The mimetype to check.
 *
 * @returns {!boolean}
 *     true if the given mimetype is supported by Guacamole.RawAudioPlayer,
 *     false otherwise.
 */
Guacamole.RawAudioPlayer.isSupportedType = function isSupportedType(mimetype) {
    // No supported types if no Web Audio API
    if (!Guacamole.AudioContextFactory.getAudioContext()) return false;

    return Guacamole.RawAudioFormat.parse(mimetype) !== null;
};

/**
 * Returns a list of all mimetypes supported by Guacamole.RawAudioPlayer. Only
 * the core mimetypes themselves will be listed. Any mimetype parameters, even
 * required ones, will not be included in the list. For example, "audio/L8" is
 * a raw audio mimetype that may be supported, but it is invalid without
 * additional parameters. Something like "audio/L8;rate=44100" would be valid,
 * however (see https://tools.ietf.org/html/rfc4856).
 *
 * @returns {!string[]}
 *     A list of all mimetypes supported by Guacamole.RawAudioPlayer, excluding
 *     any parameters. If the necessary JavaScript APIs for playing raw audio
 *     are absent, this list will be empty.
 */
Guacamole.RawAudioPlayer.getSupportedTypes = function getSupportedTypes() {
    // No supported types if no Web Audio API
    if (!Guacamole.AudioContextFactory.getAudioContext()) return [];

    // We support 8-bit and 16-bit raw PCM
    return ['audio/L8', 'audio/L16'];
};

/**
 * Abstract audio recorder which streams arbitrary audio data to an underlying
 * Guacamole.OutputStream. It is up to implementations of this class to provide
 * some means of handling this Guacamole.OutputStream. Data produced by the
 * recorder is to be sent along the provided stream immediately.
 *
 * @constructor
 */
Guacamole.AudioRecorder = function AudioRecorder() {
    /**
     * Callback which is invoked when the audio recording process has stopped
     * and the underlying Guacamole stream has been closed normally. Audio will
     * only resume recording if a new Guacamole.AudioRecorder is started. This
     * Guacamole.AudioRecorder instance MAY NOT be reused.
     *
     * @event
     */
    this.onclose = null;

    /**
     * Callback which is invoked when the audio recording process cannot
     * continue due to an error, if it has started at all. The underlying
     * Guacamole stream is automatically closed. Future attempts to record
     * audio should not be made, and this Guacamole.AudioRecorder instance
     * MAY NOT be reused.
     *
     * @event
     */
    this.onerror = null;
};

/**
 * Determines whether the given mimetype is supported by any built-in
 * implementation of Guacamole.AudioRecorder, and thus will be properly handled
 * by Guacamole.AudioRecorder.getInstance().
 *
 * @param {!string} mimetype
 *     The mimetype to check.
 *
 * @returns {!boolean}
 *     true if the given mimetype is supported by any built-in
 *     Guacamole.AudioRecorder, false otherwise.
 */
Guacamole.AudioRecorder.isSupportedType = function isSupportedType(mimetype) {
    return Guacamole.RawAudioRecorder.isSupportedType(mimetype);
};

/**
 * Returns a list of all mimetypes supported by any built-in
 * Guacamole.AudioRecorder, in rough order of priority. Beware that only the
 * core mimetypes themselves will be listed. Any mimetype parameters, even
 * required ones, will not be included in the list. For example, "audio/L8" is
 * a supported raw audio mimetype that is supported, but it is invalid without
 * additional parameters. Something like "audio/L8;rate=44100" would be valid,
 * however (see https://tools.ietf.org/html/rfc4856).
 *
 * @returns {!string[]}
 *     A list of all mimetypes supported by any built-in
 *     Guacamole.AudioRecorder, excluding any parameters.
 */
Guacamole.AudioRecorder.getSupportedTypes = function getSupportedTypes() {
    return Guacamole.RawAudioRecorder.getSupportedTypes();
};

/**
 * Returns an instance of Guacamole.AudioRecorder providing support for the
 * given audio format. If support for the given audio format is not available,
 * null is returned.
 *
 * @param {!Guacamole.OutputStream} stream
 *     The Guacamole.OutputStream to send audio data through.
 *
 * @param {!string} mimetype
 *     The mimetype of the audio data to be sent along the provided stream.
 *
 * @return {Guacamole.AudioRecorder}
 *     A Guacamole.AudioRecorder instance supporting the given mimetype and
 *     writing to the given stream, or null if support for the given mimetype
 *     is absent.
 */
Guacamole.AudioRecorder.getInstance = function getInstance(stream, mimetype) {
    // Use raw audio recorder if possible
    if (Guacamole.RawAudioRecorder.isSupportedType(mimetype)) return new Guacamole.RawAudioRecorder(stream, mimetype);

    // No support for given mimetype
    return null;
};

/**
 * Implementation of Guacamole.AudioRecorder providing support for raw PCM
 * format audio. This recorder relies only on the Web Audio API and does not
 * require any browser-level support for its audio formats.
 *
 * @constructor
 * @augments Guacamole.AudioRecorder
 * @param {!Guacamole.OutputStream} stream
 *     The Guacamole.OutputStream to write audio data to.
 *
 * @param {!string} mimetype
 *     The mimetype of the audio data to send along the provided stream, which
 *     must be a "audio/L8" or "audio/L16" mimetype with necessary parameters,
 *     such as: "audio/L16;rate=44100,channels=2".
 */
Guacamole.RawAudioRecorder = function RawAudioRecorder(stream, mimetype) {
    /**
     * Reference to this RawAudioRecorder.
     *
     * @private
     * @type {!Guacamole.RawAudioRecorder}
     */
    var recorder = this;

    /**
     * The size of audio buffer to request from the Web Audio API when
     * recording or processing audio, in sample-frames. This must be a power of
     * two between 256 and 16384 inclusive, as required by
     * AudioContext.createScriptProcessor().
     *
     * @private
     * @constant
     * @type {!number}
     */
    var BUFFER_SIZE = 2048;

    /**
     * The window size to use when applying Lanczos interpolation, commonly
     * denoted by the variable "a".
     * See: https://en.wikipedia.org/wiki/Lanczos_resampling
     *
     * @private
     * @contant
     * @type {!number}
     */
    var LANCZOS_WINDOW_SIZE = 3;

    /**
     * The format of audio this recorder will encode.
     *
     * @private
     * @type {Guacamole.RawAudioFormat}
     */
    var format = Guacamole.RawAudioFormat.parse(mimetype);

    /**
     * An instance of a Web Audio API AudioContext object, or null if the
     * Web Audio API is not supported.
     *
     * @private
     * @type {AudioContext}
     */
    var context = Guacamole.AudioContextFactory.getAudioContext();

    // Some browsers do not implement navigator.mediaDevices - this
    // shims in this functionality to ensure code compatibility.
    if (!navigator.mediaDevices) navigator.mediaDevices = {};

    // Browsers that either do not implement navigator.mediaDevices
    // at all or do not implement it completely need the getUserMedia
    // method defined.  This shims in this function by detecting
    // one of the supported legacy methods.
    if (!navigator.mediaDevices.getUserMedia)
        navigator.mediaDevices.getUserMedia = (
            navigator.getUserMedia ||
            navigator.webkitGetUserMedia ||
            navigator.mozGetUserMedia ||
            navigator.msGetUserMedia
        ).bind(navigator);

    /**
     * Guacamole.ArrayBufferWriter wrapped around the audio output stream
     * provided when this Guacamole.RawAudioRecorder was created.
     *
     * @private
     * @type {!Guacamole.ArrayBufferWriter}
     */
    var writer = new Guacamole.ArrayBufferWriter(stream);

    /**
     * The type of typed array that will be used to represent each audio packet
     * internally. This will be either Int8Array or Int16Array, depending on
     * whether the raw audio format is 8-bit or 16-bit.
     *
     * @private
     * @constructor
     */
    var SampleArray = format.bytesPerSample === 1 ? window.Int8Array : window.Int16Array;

    /**
     * The maximum absolute value of any sample within a raw audio packet sent
     * by this audio recorder. This depends only on the size of each sample,
     * and will be 128 for 8-bit audio and 32768 for 16-bit audio.
     *
     * @private
     * @type {!number}
     */
    var maxSampleValue = format.bytesPerSample === 1 ? 128 : 32768;

    /**
     * The total number of audio samples read from the local audio input device
     * over the life of this audio recorder.
     *
     * @private
     * @type {!number}
     */
    var readSamples = 0;

    /**
     * The total number of audio samples written to the underlying Guacamole
     * connection over the life of this audio recorder.
     *
     * @private
     * @type {!number}
     */
    var writtenSamples = 0;

    /**
     * The audio stream provided by the browser, if allowed. If no stream has
     * yet been received, this will be null.
     *
     * @private
     * @type {MediaStream}
     */
    var mediaStream = null;

    /**
     * The source node providing access to the local audio input device.
     *
     * @private
     * @type {MediaStreamAudioSourceNode}
     */
    var source = null;

    /**
     * The script processing node which receives audio input from the media
     * stream source node as individual audio buffers.
     *
     * @private
     * @type {ScriptProcessorNode}
     */
    var processor = null;

    /**
     * The normalized sinc function. The normalized sinc function is defined as
     * 1 for x=0 and sin(PI * x) / (PI * x) for all other values of x.
     *
     * See: https://en.wikipedia.org/wiki/Sinc_function
     *
     * @private
     * @param {!number} x
     *     The point at which the normalized sinc function should be computed.
     *
     * @returns {!number}
     *     The value of the normalized sinc function at x.
     */
    var sinc = function sinc(x) {
        // The value of sinc(0) is defined as 1
        if (x === 0) return 1;

        // Otherwise, normlized sinc(x) is sin(PI * x) / (PI * x)
        var piX = Math.PI * x;
        return Math.sin(piX) / piX;
    };

    /**
     * Calculates the value of the Lanczos kernal at point x for a given window
     * size. See: https://en.wikipedia.org/wiki/Lanczos_resampling
     *
     * @private
     * @param {!number} x
     *     The point at which the value of the Lanczos kernel should be
     *     computed.
     *
     * @param {!number} a
     *     The window size to use for the Lanczos kernel.
     *
     * @returns {!number}
     *     The value of the Lanczos kernel at the given point for the given
     *     window size.
     */
    var lanczos = function lanczos(x, a) {
        // Lanczos is sinc(x) * sinc(x / a) for -a < x < a ...
        if (-a < x && x < a) return sinc(x) * sinc(x / a);

        // ... and 0 otherwise
        return 0;
    };

    /**
     * Determines the value of the waveform represented by the audio data at
     * the given location. If the value cannot be determined exactly as it does
     * not correspond to an exact sample within the audio data, the value will
     * be derived through interpolating nearby samples.
     *
     * @private
     * @param {!Float32Array} audioData
     *     An array of audio data, as returned by AudioBuffer.getChannelData().
     *
     * @param {!number} t
     *     The relative location within the waveform from which the value
     *     should be retrieved, represented as a floating point number between
     *     0 and 1 inclusive, where 0 represents the earliest point in time and
     *     1 represents the latest.
     *
     * @returns {!number}
     *     The value of the waveform at the given location.
     */
    var interpolateSample = function getValueAt(audioData, t) {
        // Convert [0, 1] range to [0, audioData.length - 1]
        var index = (audioData.length - 1) * t;

        // Determine the start and end points for the summation used by the
        // Lanczos interpolation algorithm (see: https://en.wikipedia.org/wiki/Lanczos_resampling)
        var start = Math.floor(index) - LANCZOS_WINDOW_SIZE + 1;
        var end = Math.floor(index) + LANCZOS_WINDOW_SIZE;

        // Calculate the value of the Lanczos interpolation function for the
        // required range
        var sum = 0;
        for (var i = start; i <= end; i++) {
            sum += (audioData[i] || 0) * lanczos(index - i, LANCZOS_WINDOW_SIZE);
        }

        return sum;
    };

    /**
     * Converts the given AudioBuffer into an audio packet, ready for streaming
     * along the underlying output stream. Unlike the raw audio packets used by
     * this audio recorder, AudioBuffers require floating point samples and are
     * split into isolated planes of channel-specific data.
     *
     * @private
     * @param {!AudioBuffer} audioBuffer
     *     The Web Audio API AudioBuffer that should be converted to a raw
     *     audio packet.
     *
     * @returns {!SampleArray}
     *     A new raw audio packet containing the audio data from the provided
     *     AudioBuffer.
     */
    var toSampleArray = function toSampleArray(audioBuffer) {
        // Track overall amount of data read
        var inSamples = audioBuffer.length;
        readSamples += inSamples;

        // Calculate the total number of samples that should be written as of
        // the audio data just received and adjust the size of the output
        // packet accordingly
        var expectedWrittenSamples = Math.round((readSamples * format.rate) / audioBuffer.sampleRate);
        var outSamples = expectedWrittenSamples - writtenSamples;

        // Update number of samples written
        writtenSamples += outSamples;

        // Get array for raw PCM storage
        var data = new SampleArray(outSamples * format.channels);

        // Convert each channel
        for (var channel = 0; channel < format.channels; channel++) {
            var audioData = audioBuffer.getChannelData(channel);

            // Fill array with data from audio buffer channel
            var offset = channel;
            for (var i = 0; i < outSamples; i++) {
                data[offset] = interpolateSample(audioData, i / (outSamples - 1)) * maxSampleValue;
                offset += format.channels;
            }
        }

        return data;
    };

    /**
     * getUserMedia() callback which handles successful retrieval of an
     * audio stream (successful start of recording).
     *
     * @private
     * @param {!MediaStream} stream
     *     A MediaStream which provides access to audio data read from the
     *     user's local audio input device.
     */
    var streamReceived = function streamReceived(stream) {
        // Create processing node which receives appropriately-sized audio buffers
        processor = context.createScriptProcessor(BUFFER_SIZE, format.channels, format.channels);
        processor.connect(context.destination);

        // Send blobs when audio buffers are received
        processor.onaudioprocess = function processAudio(e) {
            writer.sendData(toSampleArray(e.inputBuffer).buffer);
        };

        // Connect processing node to user's audio input source
        source = context.createMediaStreamSource(stream);
        source.connect(processor);

        // Attempt to explicitly resume AudioContext, as it may be paused
        // by default
        if (context.state === 'suspended') context.resume();

        // Save stream for later cleanup
        mediaStream = stream;
    };

    /**
     * getUserMedia() callback which handles audio recording denial. The
     * underlying Guacamole output stream is closed, and the failure to
     * record is noted using onerror.
     *
     * @private
     */
    var streamDenied = function streamDenied() {
        // Simply end stream if audio access is not allowed
        writer.sendEnd();

        // Notify of closure
        if (recorder.onerror) recorder.onerror();
    };

    /**
     * Requests access to the user's microphone and begins capturing audio. All
     * received audio data is resampled as necessary and forwarded to the
     * Guacamole stream underlying this Guacamole.RawAudioRecorder. This
     * function must be invoked ONLY ONCE per instance of
     * Guacamole.RawAudioRecorder.
     *
     * @private
     */
    var beginAudioCapture = function beginAudioCapture() {
        // Attempt to retrieve an audio input stream from the browser
        var promise = navigator.mediaDevices.getUserMedia(
            {
                audio: true,
            },
            streamReceived,
            streamDenied
        );

        // Handle stream creation/rejection via Promise for newer versions of
        // getUserMedia()
        if (promise && promise.then) promise.then(streamReceived, streamDenied);
    };

    /**
     * Stops capturing audio, if the capture has started, freeing all associated
     * resources. If the capture has not started, this function simply ends the
     * underlying Guacamole stream.
     *
     * @private
     */
    var stopAudioCapture = function stopAudioCapture() {
        // Disconnect media source node from script processor
        if (source) source.disconnect();

        // Disconnect associated script processor node
        if (processor) processor.disconnect();

        // Stop capture
        if (mediaStream) {
            var tracks = mediaStream.getTracks();
            for (var i = 0; i < tracks.length; i++) tracks[i].stop();
        }

        // Remove references to now-unneeded components
        processor = null;
        source = null;
        mediaStream = null;

        // End stream
        writer.sendEnd();
    };

    // Once audio stream is successfully open, request and begin reading audio
    writer.onack = function audioStreamAcknowledged(status) {
        // Begin capture if successful response and not yet started
        if (status.code === Guacamole.Status.Code.SUCCESS && !mediaStream) beginAudioCapture();
        // Otherwise stop capture and cease handling any further acks
        else {
            // Stop capturing audio
            stopAudioCapture();
            writer.onack = null;

            // Notify if stream has closed normally
            if (status.code === Guacamole.Status.Code.RESOURCE_CLOSED) {
                if (recorder.onclose) recorder.onclose();
            }

            // Otherwise notify of closure due to error
            else {
                if (recorder.onerror) recorder.onerror();
            }
        }
    };
};

Guacamole.RawAudioRecorder.prototype = new Guacamole.AudioRecorder();

/**
 * Determines whether the given mimetype is supported by
 * Guacamole.RawAudioRecorder.
 *
 * @param {!string} mimetype
 *     The mimetype to check.
 *
 * @returns {!boolean}
 *     true if the given mimetype is supported by Guacamole.RawAudioRecorder,
 *     false otherwise.
 */
Guacamole.RawAudioRecorder.isSupportedType = function isSupportedType(mimetype) {
    // No supported types if no Web Audio API
    if (!Guacamole.AudioContextFactory.getAudioContext()) return false;

    return Guacamole.RawAudioFormat.parse(mimetype) !== null;
};

/**
 * Returns a list of all mimetypes supported by Guacamole.RawAudioRecorder. Only
 * the core mimetypes themselves will be listed. Any mimetype parameters, even
 * required ones, will not be included in the list. For example, "audio/L8" is
 * a raw audio mimetype that may be supported, but it is invalid without
 * additional parameters. Something like "audio/L8;rate=44100" would be valid,
 * however (see https://tools.ietf.org/html/rfc4856).
 *
 * @returns {!string[]}
 *     A list of all mimetypes supported by Guacamole.RawAudioRecorder,
 *     excluding any parameters. If the necessary JavaScript APIs for recording
 *     raw audio are absent, this list will be empty.
 */
Guacamole.RawAudioRecorder.getSupportedTypes = function getSupportedTypes() {
    // No supported types if no Web Audio API
    if (!Guacamole.AudioContextFactory.getAudioContext()) return [];

    // We support 8-bit and 16-bit raw PCM
    return ['audio/L8', 'audio/L16'];
};

/**
 * A reader which automatically handles the given input stream, assembling all
 * received blobs into a single blob by appending them to each other in order.
 * Note that this object will overwrite any installed event handlers on the
 * given Guacamole.InputStream.
 *
 * @constructor
 * @param {!Guacamole.InputStream} stream
 *     The stream that data will be read from.
 *
 * @param {!string} mimetype
 *     The mimetype of the blob being built.
 */
Guacamole.BlobReader = function (stream, mimetype) {
    /**
     * Reference to this Guacamole.InputStream.
     *
     * @private
     * @type {!Guacamole.BlobReader}
     */
    var guac_reader = this;

    /**
     * The length of this Guacamole.InputStream in bytes.
     *
     * @private
     * @type {!number}
     */
    var length = 0;

    // Get blob builder
    var blob_builder;
    if (window.BlobBuilder) blob_builder = new BlobBuilder();
    else if (window.WebKitBlobBuilder) blob_builder = new WebKitBlobBuilder();
    else if (window.MozBlobBuilder) blob_builder = new MozBlobBuilder();
    else
        blob_builder = new (function () {
            var blobs = [];

            /** @ignore */
            this.append = function (data) {
                blobs.push(new Blob([data], { type: mimetype }));
            };

            /** @ignore */
            this.getBlob = function () {
                return new Blob(blobs, { type: mimetype });
            };
        })();

    // Append received blobs
    stream.onblob = function (data) {
        // Convert to ArrayBuffer
        var binary = window.atob(data);
        var arrayBuffer = new ArrayBuffer(binary.length);
        var bufferView = new Uint8Array(arrayBuffer);

        for (var i = 0; i < binary.length; i++) bufferView[i] = binary.charCodeAt(i);

        blob_builder.append(arrayBuffer);
        length += arrayBuffer.byteLength;

        // Call handler, if present
        if (guac_reader.onprogress) guac_reader.onprogress(arrayBuffer.byteLength);

        // Send success response
        stream.sendAck('OK', 0x0000);
    };

    // Simply call onend when end received
    stream.onend = function () {
        if (guac_reader.onend) guac_reader.onend();
    };

    /**
     * Returns the current length of this Guacamole.InputStream, in bytes.
     *
     * @return {!number}
     *     The current length of this Guacamole.InputStream.
     */
    this.getLength = function () {
        return length;
    };

    /**
     * Returns the contents of this Guacamole.BlobReader as a Blob.
     *
     * @return {!Blob}
     *     The contents of this Guacamole.BlobReader.
     */
    this.getBlob = function () {
        return blob_builder.getBlob();
    };

    /**
     * Fired once for every blob of data received.
     *
     * @event
     * @param {!number} length
     *     The number of bytes received.
     */
    this.onprogress = null;

    /**
     * Fired once this stream is finished and no further data will be written.
     * @event
     */
    this.onend = null;
};

/**
 * A writer which automatically writes to the given output stream with the
 * contents of provided Blob objects.
 *
 * @constructor
 * @param {!Guacamole.OutputStream} stream
 *     The stream that data will be written to.
 */
Guacamole.BlobWriter = function BlobWriter(stream) {
    /**
     * Reference to this Guacamole.BlobWriter.
     *
     * @private
     * @type {!Guacamole.BlobWriter}
     */
    var guacWriter = this;

    /**
     * Wrapped Guacamole.ArrayBufferWriter which will be used to send any
     * provided file data.
     *
     * @private
     * @type {!Guacamole.ArrayBufferWriter}
     */
    var arrayBufferWriter = new Guacamole.ArrayBufferWriter(stream);

    // Initially, simply call onack for acknowledgements
    arrayBufferWriter.onack = function (status) {
        if (guacWriter.onack) guacWriter.onack(status);
    };

    /**
     * Browser-independent implementation of Blob.slice() which uses an end
     * offset to determine the span of the resulting slice, rather than a
     * length.
     *
     * @private
     * @param {!Blob} blob
     *     The Blob to slice.
     *
     * @param {!number} start
     *     The starting offset of the slice, in bytes, inclusive.
     *
     * @param {!number} end
     *     The ending offset of the slice, in bytes, exclusive.
     *
     * @returns {!Blob}
     *     A Blob containing the data within the given Blob starting at
     *     <code>start</code> and ending at <code>end - 1</code>.
     */
    var slice = function slice(blob, start, end) {
        // Use prefixed implementations if necessary
        var sliceImplementation = (blob.slice || blob.webkitSlice || blob.mozSlice).bind(blob);

        var length = end - start;

        // The old Blob.slice() was length-based (not end-based). Try the
        // length version first, if the two calls are not equivalent.
        if (length !== end) {
            // If the result of the slice() call matches the expected length,
            // trust that result. It must be correct.
            var sliceResult = sliceImplementation(start, length);
            if (sliceResult.size === length) return sliceResult;
        }

        // Otherwise, use the most-recent standard: end-based slice()
        return sliceImplementation(start, end);
    };

    /**
     * Sends the contents of the given blob over the underlying stream.
     *
     * @param {!Blob} blob
     *     The blob to send.
     */
    this.sendBlob = function sendBlob(blob) {
        var offset = 0;
        var reader = new FileReader();

        /**
         * Reads the next chunk of the blob provided to
         * [sendBlob()]{@link Guacamole.BlobWriter#sendBlob}. The chunk itself
         * is read asynchronously, and will not be available until
         * reader.onload fires.
         *
         * @private
         */
        var readNextChunk = function readNextChunk() {
            // If no further chunks remain, inform of completion and stop
            if (offset >= blob.size) {
                // Fire completion event for completed blob
                if (guacWriter.oncomplete) guacWriter.oncomplete(blob);

                // No further chunks to read
                return;
            }

            // Obtain reference to next chunk as a new blob
            var chunk = slice(blob, offset, offset + arrayBufferWriter.blobLength);
            offset += arrayBufferWriter.blobLength;

            // Attempt to read the blob contents represented by the blob into
            // a new array buffer
            reader.readAsArrayBuffer(chunk);
        };

        // Send each chunk over the stream, continue reading the next chunk
        reader.onload = function chunkLoadComplete() {
            // Send the successfully-read chunk
            arrayBufferWriter.sendData(reader.result);

            // Continue sending more chunks after the latest chunk is
            // acknowledged
            arrayBufferWriter.onack = function sendMoreChunks(status) {
                if (guacWriter.onack) guacWriter.onack(status);

                // Abort transfer if an error occurs
                if (status.isError()) return;

                // Inform of blob upload progress via progress events
                if (guacWriter.onprogress) guacWriter.onprogress(blob, offset - arrayBufferWriter.blobLength);

                // Queue the next chunk for reading
                readNextChunk();
            };
        };

        // If an error prevents further reading, inform of error and stop
        reader.onerror = function chunkLoadFailed() {
            // Fire error event, including the context of the error
            if (guacWriter.onerror) guacWriter.onerror(blob, offset, reader.error);
        };

        // Begin reading the first chunk
        readNextChunk();
    };

    /**
     * Signals that no further text will be sent, effectively closing the
     * stream.
     */
    this.sendEnd = function sendEnd() {
        arrayBufferWriter.sendEnd();
    };

    /**
     * Fired for received data, if acknowledged by the server.
     *
     * @event
     * @param {!Guacamole.Status} status
     *     The status of the operation.
     */
    this.onack = null;

    /**
     * Fired when an error occurs reading a blob passed to
     * [sendBlob()]{@link Guacamole.BlobWriter#sendBlob}. The transfer for the
     * the given blob will cease, but the stream will remain open.
     *
     * @event
     * @param {!Blob} blob
     *     The blob that was being read when the error occurred.
     *
     * @param {!number} offset
     *     The offset of the failed read attempt within the blob, in bytes.
     *
     * @param {!DOMError} error
     *     The error that occurred.
     */
    this.onerror = null;

    /**
     * Fired for each successfully-read chunk of data as a blob is being sent
     * via [sendBlob()]{@link Guacamole.BlobWriter#sendBlob}.
     *
     * @event
     * @param {!Blob} blob
     *     The blob that is being read.
     *
     * @param {!number} offset
     *     The offset of the read that just succeeded.
     */
    this.onprogress = null;

    /**
     * Fired when a blob passed to
     * [sendBlob()]{@link Guacamole.BlobWriter#sendBlob} has finished being
     * sent.
     *
     * @event
     * @param {!Blob} blob
     *     The blob that was sent.
     */
    this.oncomplete = null;
};

/**
 * Guacamole protocol client. Given a {@link Guacamole.Tunnel},
 * automatically handles incoming and outgoing Guacamole instructions via the
 * provided tunnel, updating its display using one or more canvas elements.
 *
 * @constructor
 * @param {!Guacamole.Tunnel} tunnel
 *     The tunnel to use to send and receive Guacamole instructions.
 */
Guacamole.Client = function (tunnel) {
    var guac_client = this;

    var currentState = Guacamole.Client.State.IDLE;

    var currentTimestamp = 0;

    /**
     * The rough number of milliseconds to wait between sending keep-alive
     * pings. This may vary depending on how frequently the browser allows
     * timers to run, as well as how frequently the client receives messages
     * from the server.
     *
     * @private
     * @constant
     * @type {!number}
     */
    var KEEP_ALIVE_FREQUENCY = 5000;

    /**
     * The current keep-alive ping timeout ID, if any. This will only be set
     * upon connecting.
     *
     * @private
     * @type {number}
     */
    var keepAliveTimeout = null;

    /**
     * The timestamp of the point in time that the last keep-live ping was
     * sent, in milliseconds elapsed since midnight of January 1, 1970 UTC.
     *
     * @private
     * @type {!number}
     */
    var lastSentKeepAlive = 0;

    /**
     * Translation from Guacamole protocol line caps to Layer line caps.
     *
     * @private
     * @type {!Object.<number, string>}
     */
    var lineCap = {
        0: 'butt',
        1: 'round',
        2: 'square',
    };

    /**
     * Translation from Guacamole protocol line caps to Layer line caps.
     *
     * @private
     * @type {!Object.<number, string>}
     */
    var lineJoin = {
        0: 'bevel',
        1: 'miter',
        2: 'round',
    };

    /**
     * The underlying Guacamole display.
     *
     * @private
     * @type {!Guacamole.Display}
     */
    var display = new Guacamole.Display();

    /**
     * All available layers and buffers
     *
     * @private
     * @type {!Object.<number, (Guacamole.Display.VisibleLayer|Guacamole.Layer)>}
     */
    var layers = {};

    /**
     * All audio players currently in use by the client. Initially, this will
     * be empty, but audio players may be allocated by the server upon request.
     *
     * @private
     * @type {!Object.<number, Guacamole.AudioPlayer>}
     */
    var audioPlayers = {};

    /**
     * All video players currently in use by the client. Initially, this will
     * be empty, but video players may be allocated by the server upon request.
     *
     * @private
     * @type {!Object.<number, Guacamole.VideoPlayer>}
     */
    var videoPlayers = {};

    // No initial parsers
    var parsers = [];

    // No initial streams
    var streams = [];

    /**
     * All current objects. The index of each object is dictated by the
     * Guacamole server.
     *
     * @private
     * @type {!Guacamole.Object[]}
     */
    var objects = [];

    // Pool of available stream indices
    var stream_indices = new Guacamole.IntegerPool();

    // Array of allocated output streams by index
    var output_streams = [];

    function setState(state) {
        if (state != currentState) {
            currentState = state;
            if (guac_client.onstatechange) guac_client.onstatechange(currentState);
        }
    }

    function isConnected() {
        return currentState == Guacamole.Client.State.CONNECTED || currentState == Guacamole.Client.State.WAITING;
    }

    /**
     * Produces an opaque representation of Guacamole.Client state which can be
     * later imported through a call to importState(). This object is
     * effectively an independent, compressed snapshot of protocol and display
     * state. Invoking this function implicitly flushes the display.
     *
     * @param {!function} callback
     *     Callback which should be invoked once the state object is ready. The
     *     state object will be passed to the callback as the sole parameter.
     *     This callback may be invoked immediately, or later as the display
     *     finishes rendering and becomes ready.
     */
    this.exportState = function exportState(callback) {
        // Start with empty state
        var state = {
            currentState: currentState,
            currentTimestamp: currentTimestamp,
            layers: {},
        };

        var layersSnapshot = {};

        // Make a copy of all current layers (protocol state)
        for (var key in layers) {
            layersSnapshot[key] = layers[key];
        }

        // Populate layers once data is available (display state, requires flush)
        display.flush(function populateLayers() {
            // Export each defined layer/buffer
            for (var key in layersSnapshot) {
                var index = parseInt(key);
                var layer = layersSnapshot[key];
                var canvas = layer.toCanvas();

                // Store layer/buffer dimensions
                var exportLayer = {
                    width: layer.width,
                    height: layer.height,
                };

                // Store layer/buffer image data, if it can be generated
                if (layer.width && layer.height) exportLayer.url = canvas.toDataURL('image/png');

                // Add layer properties if not a buffer nor the default layer
                if (index > 0) {
                    exportLayer.x = layer.x;
                    exportLayer.y = layer.y;
                    exportLayer.z = layer.z;
                    exportLayer.alpha = layer.alpha;
                    exportLayer.matrix = layer.matrix;
                    exportLayer.parent = getLayerIndex(layer.parent);
                }

                // Store exported layer
                state.layers[key] = exportLayer;
            }

            // Invoke callback now that the state is ready
            callback(state);
        });
    };

    /**
     * Restores Guacamole.Client protocol and display state based on an opaque
     * object from a prior call to exportState(). The Guacamole.Client instance
     * used to export that state need not be the same as this instance.
     *
     * @param {!object} state
     *     An opaque representation of Guacamole.Client state from a prior call
     *     to exportState().
     *
     * @param {function} [callback]
     *     The function to invoke when state has finished being imported. This
     *     may happen immediately, or later as images within the provided state
     *     object are loaded.
     */
    this.importState = function importState(state, callback) {
        var key;
        var index;

        currentState = state.currentState;
        currentTimestamp = state.currentTimestamp;

        // Cancel any pending display operations/frames
        display.cancel();

        // Dispose of all layers
        for (key in layers) {
            index = parseInt(key);
            if (index > 0) layers[key].dispose();
        }

        layers = {};

        // Import state of each layer/buffer
        for (key in state.layers) {
            index = parseInt(key);

            var importLayer = state.layers[key];
            var layer = getLayer(index);

            // Reset layer size
            display.resize(layer, importLayer.width, importLayer.height);

            // Initialize new layer if it has associated data
            if (importLayer.url) {
                display.setChannelMask(layer, Guacamole.Layer.SRC);
                display.draw(layer, 0, 0, importLayer.url);
            }

            // Set layer-specific properties if not a buffer nor the default layer
            if (index > 0 && importLayer.parent >= 0) {
                // Apply layer position and set parent
                var parent = getLayer(importLayer.parent);
                display.move(layer, parent, importLayer.x, importLayer.y, importLayer.z);

                // Set layer transparency
                display.shade(layer, importLayer.alpha);

                // Apply matrix transform
                var matrix = importLayer.matrix;
                display.distort(layer, matrix[0], matrix[1], matrix[2], matrix[3], matrix[4], matrix[5]);
            }
        }

        // Flush changes to display
        display.flush(callback);
    };

    /**
     * Returns the underlying display of this Guacamole.Client. The display
     * contains an Element which can be added to the DOM, causing the
     * display to become visible.
     *
     * @return {!Guacamole.Display}
     *     The underlying display of this Guacamole.Client.
     */
    this.getDisplay = function () {
        return display;
    };

    /**
     * Sends the current size of the screen.
     *
     * @param {!number} width
     *     The width of the screen.
     *
     * @param {!number} height
     *     The height of the screen.
     */
    this.sendSize = function (width, height) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        tunnel.sendMessage('size', width, height);
    };

    /**
     * Sends a key event having the given properties as if the user
     * pressed or released a key.
     *
     * @param {!boolean} pressed
     *     Whether the key is pressed (true) or released (false).
     *
     * @param {!number} keysym
     *     The keysym of the key being pressed or released.
     */
    this.sendKeyEvent = function (pressed, keysym) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        tunnel.sendMessage('key', keysym, pressed);
    };

    /**
     * Sends a mouse event having the properties provided by the given mouse
     * state.
     *
     * @param {!Guacamole.Mouse.State} mouseState
     *     The state of the mouse to send in the mouse event.
     *
     * @param {boolean} [applyDisplayScale=false]
     *     Whether the provided mouse state uses local display units, rather
     *     than remote display units, and should be scaled to match the
     *     {@link Guacamole.Display}.
     */
    this.sendMouseState = function sendMouseState(mouseState, applyDisplayScale) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        var x = mouseState.x;
        var y = mouseState.y;

        // Translate for display units if requested
        if (applyDisplayScale) {
            x /= display.getScale();
            y /= display.getScale();
        }

        // Update client-side cursor
        display.moveCursor(Math.floor(x), Math.floor(y));

        // Build mask
        var buttonMask = 0;
        if (mouseState.left) buttonMask |= 1;
        if (mouseState.middle) buttonMask |= 2;
        if (mouseState.right) buttonMask |= 4;
        if (mouseState.up) buttonMask |= 8;
        if (mouseState.down) buttonMask |= 16;

        // Send message
        tunnel.sendMessage('mouse', Math.floor(x), Math.floor(y), buttonMask);
    };

    /**
     * Sends a touch event having the properties provided by the given touch
     * state.
     *
     * @param {!Guacamole.Touch.State} touchState
     *     The state of the touch contact to send in the touch event.
     *
     * @param {boolean} [applyDisplayScale=false]
     *     Whether the provided touch state uses local display units, rather
     *     than remote display units, and should be scaled to match the
     *     {@link Guacamole.Display}.
     */
    this.sendTouchState = function sendTouchState(touchState, applyDisplayScale) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        var x = touchState.x;
        var y = touchState.y;

        // Translate for display units if requested
        if (applyDisplayScale) {
            x /= display.getScale();
            y /= display.getScale();
        }

        tunnel.sendMessage(
            'touch',
            touchState.id,
            Math.floor(x),
            Math.floor(y),
            Math.floor(touchState.radiusX),
            Math.floor(touchState.radiusY),
            touchState.angle,
            touchState.force
        );
    };

    /**
     * Allocates an available stream index and creates a new
     * Guacamole.OutputStream using that index, associating the resulting
     * stream with this Guacamole.Client. Note that this stream will not yet
     * exist as far as the other end of the Guacamole connection is concerned.
     * Streams exist within the Guacamole protocol only when referenced by an
     * instruction which creates the stream, such as a "clipboard", "file", or
     * "pipe" instruction.
     *
     * @returns {!Guacamole.OutputStream}
     *     A new Guacamole.OutputStream with a newly-allocated index and
     *     associated with this Guacamole.Client.
     */
    this.createOutputStream = function createOutputStream() {
        // Allocate index
        var index = stream_indices.next();

        // Return new stream
        var stream = (output_streams[index] = new Guacamole.OutputStream(guac_client, index));
        return stream;
    };

    /**
     * Opens a new audio stream for writing, where audio data having the give
     * mimetype will be sent along the returned stream. The instruction
     * necessary to create this stream will automatically be sent.
     *
     * @param {!string} mimetype
     *     The mimetype of the audio data that will be sent along the returned
     *     stream.
     *
     * @return {!Guacamole.OutputStream}
     *     The created audio stream.
     */
    this.createAudioStream = function (mimetype) {
        // Allocate and associate stream with audio metadata
        var stream = guac_client.createOutputStream();
        tunnel.sendMessage('audio', stream.index, mimetype);
        return stream;
    };

    /**
     * Opens a new file for writing, having the given index, mimetype and
     * filename. The instruction necessary to create this stream will
     * automatically be sent.
     *
     * @param {!string} mimetype
     *     The mimetype of the file being sent.
     *
     * @param {!string} filename
     *     The filename of the file being sent.
     *
     * @return {!Guacamole.OutputStream}
     *     The created file stream.
     */
    this.createFileStream = function (mimetype, filename) {
        // Allocate and associate stream with file metadata
        var stream = guac_client.createOutputStream();
        tunnel.sendMessage('file', stream.index, mimetype, filename);
        return stream;
    };

    /**
     * Opens a new pipe for writing, having the given name and mimetype. The
     * instruction necessary to create this stream will automatically be sent.
     *
     * @param {!string} mimetype
     *     The mimetype of the data being sent.
     *
     * @param {!string} name
     *     The name of the pipe.
     *
     * @return {!Guacamole.OutputStream}
     *     The created file stream.
     */
    this.createPipeStream = function (mimetype, name) {
        // Allocate and associate stream with pipe metadata
        var stream = guac_client.createOutputStream();
        tunnel.sendMessage('pipe', stream.index, mimetype, name);
        return stream;
    };

    /**
     * Opens a new clipboard object for writing, having the given mimetype. The
     * instruction necessary to create this stream will automatically be sent.
     *
     * @param {!string} mimetype
     *     The mimetype of the data being sent.
     *
     * @param {!string} name
     *     The name of the pipe.
     *
     * @return {!Guacamole.OutputStream}
     *     The created file stream.
     */
    this.createClipboardStream = function (mimetype) {
        // Allocate and associate stream with clipboard metadata
        var stream = guac_client.createOutputStream();
        tunnel.sendMessage('clipboard', stream.index, mimetype);
        return stream;
    };

    /**
     * Opens a new argument value stream for writing, having the given
     * parameter name and mimetype, requesting that the connection parameter
     * with the given name be updated to the value described by the contents
     * of the following stream. The instruction necessary to create this stream
     * will automatically be sent.
     *
     * @param {!string} mimetype
     *     The mimetype of the data being sent.
     *
     * @param {!string} name
     *     The name of the connection parameter to attempt to update.
     *
     * @return {!Guacamole.OutputStream}
     *     The created argument value stream.
     */
    this.createArgumentValueStream = function createArgumentValueStream(mimetype, name) {
        // Allocate and associate stream with argument value metadata
        var stream = guac_client.createOutputStream();
        tunnel.sendMessage('argv', stream.index, mimetype, name);
        return stream;
    };

    /**
     * Creates a new output stream associated with the given object and having
     * the given mimetype and name. The legality of a mimetype and name is
     * dictated by the object itself. The instruction necessary to create this
     * stream will automatically be sent.
     *
     * @param {!number} index
     *     The index of the object for which the output stream is being
     *     created.
     *
     * @param {!string} mimetype
     *     The mimetype of the data which will be sent to the output stream.
     *
     * @param {!string} name
     *     The defined name of an output stream within the given object.
     *
     * @returns {!Guacamole.OutputStream}
     *     An output stream which will write blobs to the named output stream
     *     of the given object.
     */
    this.createObjectOutputStream = function createObjectOutputStream(index, mimetype, name) {
        // Allocate and associate stream with object metadata
        var stream = guac_client.createOutputStream();
        tunnel.sendMessage('put', index, stream.index, mimetype, name);
        return stream;
    };

    /**
     * Requests read access to the input stream having the given name. If
     * successful, a new input stream will be created.
     *
     * @param {!number} index
     *     The index of the object from which the input stream is being
     *     requested.
     *
     * @param {!string} name
     *     The name of the input stream to request.
     */
    this.requestObjectInputStream = function requestObjectInputStream(index, name) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        tunnel.sendMessage('get', index, name);
    };

    /**
     * Acknowledge receipt of a blob on the stream with the given index.
     *
     * @param {!number} index
     *     The index of the stream associated with the received blob.
     *
     * @param {!string} message
     *     A human-readable message describing the error or status.
     *
     * @param {!number} code
     *     The error code, if any, or 0 for success.
     */
    this.sendAck = function (index, message, code) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        tunnel.sendMessage('ack', index, message, code);
    };

    /**
     * Given the index of a file, writes a blob of data to that file.
     *
     * @param {!number} index
     *     The index of the file to write to.
     *
     * @param {!string} data
     *     Base64-encoded data to write to the file.
     */
    this.sendBlob = function (index, data) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        tunnel.sendMessage('blob', index, data);
    };

    /**
     * Marks a currently-open stream as complete. The other end of the
     * Guacamole connection will be notified via an "end" instruction that the
     * stream is closed, and the index will be made available for reuse in
     * future streams.
     *
     * @param {!number} index
     *     The index of the stream to end.
     */
    this.endStream = function (index) {
        // Do not send requests if not connected
        if (!isConnected()) return;

        // Explicitly close stream by sending "end" instruction
        tunnel.sendMessage('end', index);

        // Free associated index and stream if they exist
        if (output_streams[index]) {
            stream_indices.free(index);
            delete output_streams[index];
        }
    };

    /**
     * Fired whenever the state of this Guacamole.Client changes.
     *
     * @event
     * @param {!number} state
     *     The new state of the client.
     */
    this.onstatechange = null;

    /**
     * Fired when the remote client sends a name update.
     *
     * @event
     * @param {!string} name
     *     The new name of this client.
     */
    this.onname = null;

    /**
     * Fired when an error is reported by the remote client, and the connection
     * is being closed.
     *
     * @event
     * @param {!Guacamole.Status} status
     *     A status object which describes the error.
     */
    this.onerror = null;

    /**
     * Fired when an arbitrary message is received from the tunnel that should
     * be processed by the client. By default, additional message-specific
     * events such as "onjoin" and "onleave" will fire for the received message
     * after this event has been processed. An event handler for "onmsg" need
     * not be supplied if "onjoin" and/or "onleave" will be used.
     *
     * @event
     * @param {!number} msgcode
     *     A status code sent by the remote server that indicates the nature of
     *     the message that is being sent to the client.
     *
     * @param {string[]} args
     *     An array of arguments to be processed with the message sent to the
     *     client.
     *
     * @return {boolean}
     *     true if message-specific events such as "onjoin" and
     *     "onleave" should be fired for this message, false otherwise. If
     *     no value is returned, message-specific events will be allowed to
     *     fire.
     */
    this.onmsg = null;

    /**
     * Fired when a user joins a shared connection.
     *
     * @event
     * @param {!string} userID
     *     A unique value representing this specific user's connection to the
     *     shared connection. This value is generated by the server and is
     *     guaranteed to be unique relative to other users of the connection.
     *
     * @param {!string} name
     *     A human-readable name representing the user that joined, such as
     *     their username. This value is provided by the web application during
     *     the connection handshake and is not necessarily unique relative to
     *     other users of the connection.
     */
    this.onjoin = null;

    /**
     * Fired when a user leaves a shared connection.
     *
     * @event
     * @param {!string} userID
     *     A unique value representing this specific user's connection to the
     *     shared connection. This value is generated by the server and is
     *     guaranteed to be unique relative to other users of the connection.
     *
     * @param {!string} name
     *     A human-readable name representing the user that left, such as their
     *     username. This value is provided by the web application during the
     *     connection handshake and is not necessarily unique relative to other
     *     users of the connection.
     */
    this.onleave = null;

    /**
     * Fired when a audio stream is created. The stream provided to this event
     * handler will contain its own event handlers for received data.
     *
     * @event
     * @param {!Guacamole.InputStream} stream
     *     The stream that will receive audio data from the server.
     *
     * @param {!string} mimetype
     *     The mimetype of the audio data which will be received.
     *
     * @return {Guacamole.AudioPlayer}
     *     An object which implements the Guacamole.AudioPlayer interface and
     *     has been initialized to play the data in the provided stream, or null
     *     if the built-in audio players of the Guacamole client should be
     *     used.
     */
    this.onaudio = null;

    /**
     * Fired when a video stream is created. The stream provided to this event
     * handler will contain its own event handlers for received data.
     *
     * @event
     * @param {!Guacamole.InputStream} stream
     *     The stream that will receive video data from the server.
     *
     * @param {!Guacamole.Display.VisibleLayer} layer
     *     The destination layer on which the received video data should be
     *     played. It is the responsibility of the Guacamole.VideoPlayer
     *     implementation to play the received data within this layer.
     *
     * @param {!string} mimetype
     *     The mimetype of the video data which will be received.
     *
     * @return {Guacamole.VideoPlayer}
     *     An object which implements the Guacamole.VideoPlayer interface and
     *     has been initialized to play the data in the provided stream, or null
     *     if the built-in video players of the Guacamole client should be
     *     used.
     */
    this.onvideo = null;

    /**
     * Fired when the remote client is explicitly declaring the level of
     * multi-touch support provided by a particular display layer.
     *
     * @event
     * @param {!Guacamole.Display.VisibleLayer} layer
     *     The layer whose multi-touch support level is being declared.
     *
     * @param {!number} touches
     *     The maximum number of simultaneous touches supported by the given
     *     layer, where 0 indicates that touch events are not supported at all.
     */
    this.onmultitouch = null;

    /**
     * Fired when the current value of a connection parameter is being exposed
     * by the server.
     *
     * @event
     * @param {!Guacamole.InputStream} stream
     *     The stream that will receive connection parameter data from the
     *     server.
     *
     * @param {!string} mimetype
     *     The mimetype of the data which will be received.
     *
     * @param {!string} name
     *     The name of the connection parameter whose value is being exposed.
     */
    this.onargv = null;

    /**
     * Fired when the clipboard of the remote client is changing.
     *
     * @event
     * @param {!Guacamole.InputStream} stream
     *     The stream that will receive clipboard data from the server.
     *
     * @param {!string} mimetype
     *     The mimetype of the data which will be received.
     */
    this.onclipboard = null;

    /**
     * Fired when a file stream is created. The stream provided to this event
     * handler will contain its own event handlers for received data.
     *
     * @event
     * @param {!Guacamole.InputStream} stream
     *     The stream that will receive data from the server.
     *
     * @param {!string} mimetype
     *     The mimetype of the file received.
     *
     * @param {!string} filename
     *     The name of the file received.
     */
    this.onfile = null;

    /**
     * Fired when a filesystem object is created. The object provided to this
     * event handler will contain its own event handlers and functions for
     * requesting and handling data.
     *
     * @event
     * @param {!Guacamole.Object} object
     *     The created filesystem object.
     *
     * @param {!string} name
     *     The name of the filesystem.
     */
    this.onfilesystem = null;

    /**
     * Fired when a pipe stream is created. The stream provided to this event
     * handler will contain its own event handlers for received data;
     *
     * @event
     * @param {!Guacamole.InputStream} stream
     *     The stream that will receive data from the server.
     *
     * @param {!string} mimetype
     *     The mimetype of the data which will be received.
     *
     * @param {!string} name
     *     The name of the pipe.
     */
    this.onpipe = null;

    /**
     * Fired when a "required" instruction is received. A required instruction
     * indicates that additional parameters are required for the connection to
     * continue, such as user credentials.
     *
     * @event
     * @param {!string[]} parameters
     *      The names of the connection parameters that are required to be
     *      provided for the connection to continue.
     */
    this.onrequired = null;

    /**
     * Fired whenever a sync instruction is received from the server, indicating
     * that the server is finished processing any input from the client and
     * has sent any results.
     *
     * @event
     * @param {!number} timestamp
     *     The timestamp associated with the sync instruction.
     *
     * @param {!number} frames
     *     The number of frames that were considered or combined to produce the
     *     frame associated with this sync instruction, or zero if this value
     *     is not known or the remote desktop server provides no concept of
     *     frames.
     */
    this.onsync = null;

    /**
     * Returns the layer with the given index, creating it if necessary.
     * Positive indices refer to visible layers, an index of zero refers to
     * the default layer, and negative indices refer to buffers.
     *
     * @private
     * @param {!number} index
     *     The index of the layer to retrieve.
     *
     * @return {!(Guacamole.Display.VisibleLayer|Guacamole.Layer)}
     *     The layer having the given index.
     */
    var getLayer = function getLayer(index) {
        // Get layer, create if necessary
        var layer = layers[index];
        if (!layer) {
            // Create layer based on index
            if (index === 0) layer = display.getDefaultLayer();
            else if (index > 0) layer = display.createLayer();
            else layer = display.createBuffer();

            // Add new layer
            layers[index] = layer;
        }

        return layer;
    };

    /**
     * Returns the index passed to getLayer() when the given layer was created.
     * Positive indices refer to visible layers, an index of zero refers to the
     * default layer, and negative indices refer to buffers.
     *
     * @param {!(Guacamole.Display.VisibleLayer|Guacamole.Layer)} layer
     *     The layer whose index should be determined.
     *
     * @returns {number}
     *     The index of the given layer, or null if no such layer is associated
     *     with this client.
     */
    var getLayerIndex = function getLayerIndex(layer) {
        // Avoid searching if there clearly is no such layer
        if (!layer) return null;

        // Search through each layer, returning the index of the given layer
        // once found
        for (var key in layers) {
            if (layer === layers[key]) return parseInt(key);
        }

        // Otherwise, no such index
        return null;
    };

    function getParser(index) {
        var parser = parsers[index];

        // If parser not yet created, create it, and tie to the
        // oninstruction handler of the tunnel.
        if (parser == null) {
            parser = parsers[index] = new Guacamole.Parser();
            parser.oninstruction = tunnel.oninstruction;
        }

        return parser;
    }

    /**
     * Handlers for all defined layer properties.
     *
     * @private
     * @type {!Object.<string, function>}
     */
    var layerPropertyHandlers = {
        'miter-limit': function (layer, value) {
            display.setMiterLimit(layer, parseFloat(value));
        },

        'multi-touch': function layerSupportsMultiTouch(layer, value) {
            // Process "multi-touch" property only for true visible layers (not off-screen buffers)
            if (guac_client.onmultitouch && layer instanceof Guacamole.Display.VisibleLayer) guac_client.onmultitouch(layer, parseInt(value));
        },
    };

    /**
     * Handlers for all instruction opcodes receivable by a Guacamole protocol
     * client.
     *
     * @private
     * @type {!Object.<string, function>}
     */
    var instructionHandlers = {
        ack: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var reason = parameters[1];
            var code = parseInt(parameters[2]);

            // Get stream
            var stream = output_streams[stream_index];
            if (stream) {
                // Signal ack if handler defined
                if (stream.onack) stream.onack(new Guacamole.Status(code, reason));

                // If code is an error, invalidate stream if not already
                // invalidated by onack handler
                if (code >= 0x0100 && output_streams[stream_index] === stream) {
                    stream_indices.free(stream_index);
                    delete output_streams[stream_index];
                }
            }
        },

        arc: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var x = parseInt(parameters[1]);
            var y = parseInt(parameters[2]);
            var radius = parseInt(parameters[3]);
            var startAngle = parseFloat(parameters[4]);
            var endAngle = parseFloat(parameters[5]);
            var negative = parseInt(parameters[6]);

            display.arc(layer, x, y, radius, startAngle, endAngle, negative != 0);
        },

        argv: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var mimetype = parameters[1];
            var name = parameters[2];

            // Create stream
            if (guac_client.onargv) {
                var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));
                guac_client.onargv(stream, mimetype, name);
            }

            // Otherwise, unsupported
            else guac_client.sendAck(stream_index, 'Receiving argument values unsupported', 0x0100);
        },

        audio: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var mimetype = parameters[1];

            // Create stream
            var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));

            // Get player instance via callback
            var audioPlayer = null;
            if (guac_client.onaudio) audioPlayer = guac_client.onaudio(stream, mimetype);

            // If unsuccessful, try to use a default implementation
            if (!audioPlayer) audioPlayer = Guacamole.AudioPlayer.getInstance(stream, mimetype);

            // If we have successfully retrieved an audio player, send success response
            if (audioPlayer) {
                audioPlayers[stream_index] = audioPlayer;
                guac_client.sendAck(stream_index, 'OK', 0x0000);
            }

            // Otherwise, mimetype must be unsupported
            else guac_client.sendAck(stream_index, 'BAD TYPE', 0x030f);
        },

        blob: function (parameters) {
            // Get stream
            var stream_index = parseInt(parameters[0]);
            var data = parameters[1];
            var stream = streams[stream_index];

            // Write data
            if (stream && stream.onblob) stream.onblob(data);
        },

        body: function handleBody(parameters) {
            // Get object
            var objectIndex = parseInt(parameters[0]);
            var object = objects[objectIndex];

            var streamIndex = parseInt(parameters[1]);
            var mimetype = parameters[2];
            var name = parameters[3];

            // Create stream if handler defined
            if (object && object.onbody) {
                var stream = (streams[streamIndex] = new Guacamole.InputStream(guac_client, streamIndex));
                object.onbody(stream, mimetype, name);
            }

            // Otherwise, unsupported
            else guac_client.sendAck(streamIndex, 'Receipt of body unsupported', 0x0100);
        },

        cfill: function (parameters) {
            var channelMask = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var r = parseInt(parameters[2]);
            var g = parseInt(parameters[3]);
            var b = parseInt(parameters[4]);
            var a = parseInt(parameters[5]);

            display.setChannelMask(layer, channelMask);
            display.fillColor(layer, r, g, b, a);
        },

        clip: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));

            display.clip(layer);
        },

        clipboard: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var mimetype = parameters[1];

            // Create stream
            if (guac_client.onclipboard) {
                var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));
                guac_client.onclipboard(stream, mimetype);
            }

            // Otherwise, unsupported
            else guac_client.sendAck(stream_index, 'Clipboard unsupported', 0x0100);
        },

        close: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));

            display.close(layer);
        },

        copy: function (parameters) {
            var srcL = getLayer(parseInt(parameters[0]));
            var srcX = parseInt(parameters[1]);
            var srcY = parseInt(parameters[2]);
            var srcWidth = parseInt(parameters[3]);
            var srcHeight = parseInt(parameters[4]);
            var channelMask = parseInt(parameters[5]);
            var dstL = getLayer(parseInt(parameters[6]));
            var dstX = parseInt(parameters[7]);
            var dstY = parseInt(parameters[8]);

            display.setChannelMask(dstL, channelMask);
            display.copy(srcL, srcX, srcY, srcWidth, srcHeight, dstL, dstX, dstY);
        },

        cstroke: function (parameters) {
            var channelMask = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var cap = lineCap[parseInt(parameters[2])];
            var join = lineJoin[parseInt(parameters[3])];
            var thickness = parseInt(parameters[4]);
            var r = parseInt(parameters[5]);
            var g = parseInt(parameters[6]);
            var b = parseInt(parameters[7]);
            var a = parseInt(parameters[8]);

            display.setChannelMask(layer, channelMask);
            display.strokeColor(layer, cap, join, thickness, r, g, b, a);
        },

        cursor: function (parameters) {
            var cursorHotspotX = parseInt(parameters[0]);
            var cursorHotspotY = parseInt(parameters[1]);
            var srcL = getLayer(parseInt(parameters[2]));
            var srcX = parseInt(parameters[3]);
            var srcY = parseInt(parameters[4]);
            var srcWidth = parseInt(parameters[5]);
            var srcHeight = parseInt(parameters[6]);

            display.setCursor(cursorHotspotX, cursorHotspotY, srcL, srcX, srcY, srcWidth, srcHeight);
        },

        curve: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var cp1x = parseInt(parameters[1]);
            var cp1y = parseInt(parameters[2]);
            var cp2x = parseInt(parameters[3]);
            var cp2y = parseInt(parameters[4]);
            var x = parseInt(parameters[5]);
            var y = parseInt(parameters[6]);

            display.curveTo(layer, cp1x, cp1y, cp2x, cp2y, x, y);
        },

        disconnect: function handleDisconnect(parameters) {
            // Explicitly tear down connection
            guac_client.disconnect();
        },

        dispose: function (parameters) {
            var layer_index = parseInt(parameters[0]);

            // If visible layer, remove from parent
            if (layer_index > 0) {
                // Remove from parent
                var layer = getLayer(layer_index);
                display.dispose(layer);

                // Delete reference
                delete layers[layer_index];
            }

            // If buffer, just delete reference
            else if (layer_index < 0) delete layers[layer_index];

            // Attempting to dispose the root layer currently has no effect.
        },

        distort: function (parameters) {
            var layer_index = parseInt(parameters[0]);
            var a = parseFloat(parameters[1]);
            var b = parseFloat(parameters[2]);
            var c = parseFloat(parameters[3]);
            var d = parseFloat(parameters[4]);
            var e = parseFloat(parameters[5]);
            var f = parseFloat(parameters[6]);

            // Only valid for visible layers (not buffers)
            if (layer_index >= 0) {
                var layer = getLayer(layer_index);
                display.distort(layer, a, b, c, d, e, f);
            }
        },

        error: function (parameters) {
            var reason = parameters[0];
            var code = parseInt(parameters[1]);

            // Call handler if defined
            if (guac_client.onerror) guac_client.onerror(new Guacamole.Status(code, reason));

            guac_client.disconnect();
        },

        end: function (parameters) {
            var stream_index = parseInt(parameters[0]);

            // Get stream
            var stream = streams[stream_index];
            if (stream) {
                // Signal end of stream if handler defined
                if (stream.onend) stream.onend();

                // Invalidate stream
                delete streams[stream_index];
            }
        },

        file: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var mimetype = parameters[1];
            var filename = parameters[2];

            // Create stream
            if (guac_client.onfile) {
                var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));
                guac_client.onfile(stream, mimetype, filename);
            }

            // Otherwise, unsupported
            else guac_client.sendAck(stream_index, 'File transfer unsupported', 0x0100);
        },

        filesystem: function handleFilesystem(parameters) {
            var objectIndex = parseInt(parameters[0]);
            var name = parameters[1];

            // Create object, if supported
            if (guac_client.onfilesystem) {
                var object = (objects[objectIndex] = new Guacamole.Object(guac_client, objectIndex));
                guac_client.onfilesystem(object, name);
            }

            // If unsupported, simply ignore the availability of the filesystem
        },

        identity: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));

            display.setTransform(layer, 1, 0, 0, 1, 0, 0);
        },

        img: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var channelMask = parseInt(parameters[1]);
            var layer = getLayer(parseInt(parameters[2]));
            var mimetype = parameters[3];
            var x = parseInt(parameters[4]);
            var y = parseInt(parameters[5]);

            // Create stream
            var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));

            // Draw received contents once decoded
            display.setChannelMask(layer, channelMask);
            display.drawStream(layer, x, y, stream, mimetype);
        },

        jpeg: function (parameters) {
            var channelMask = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var x = parseInt(parameters[2]);
            var y = parseInt(parameters[3]);
            var data = parameters[4];

            display.setChannelMask(layer, channelMask);
            display.draw(layer, x, y, 'data:image/jpeg;base64,' + data);
        },

        lfill: function (parameters) {
            var channelMask = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var srcLayer = getLayer(parseInt(parameters[2]));

            display.setChannelMask(layer, channelMask);
            display.fillLayer(layer, srcLayer);
        },

        line: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var x = parseInt(parameters[1]);
            var y = parseInt(parameters[2]);

            display.lineTo(layer, x, y);
        },

        lstroke: function (parameters) {
            var channelMask = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var srcLayer = getLayer(parseInt(parameters[2]));

            display.setChannelMask(layer, channelMask);
            display.strokeLayer(layer, srcLayer);
        },

        mouse: function handleMouse(parameters) {
            var x = parseInt(parameters[0]);
            var y = parseInt(parameters[1]);

            // Display and move software cursor to received coordinates
            display.showCursor(true);
            display.moveCursor(x, y);
        },

        move: function (parameters) {
            var layer_index = parseInt(parameters[0]);
            var parent_index = parseInt(parameters[1]);
            var x = parseInt(parameters[2]);
            var y = parseInt(parameters[3]);
            var z = parseInt(parameters[4]);

            // Only valid for non-default layers
            if (layer_index > 0 && parent_index >= 0) {
                var layer = getLayer(layer_index);
                var parent = getLayer(parent_index);
                display.move(layer, parent, x, y, z);
            }
        },

        msg: function (parameters) {
            var userID;
            var username;

            // Fire general message handling event first
            var allowDefault = true;
            var msgid = parseInt(parameters[0]);
            if (guac_client.onmsg) {
                allowDefault = guac_client.onmsg(msgid, parameters.slice(1));
                if (allowDefault === undefined) allowDefault = true;
            }

            // Fire message-specific convenience events if not prevented by the
            // "onmsg" handler
            if (allowDefault) {
                switch (msgid) {
                    case Guacamole.Client.Message.USER_JOINED:
                        userID = parameters[1];
                        username = parameters[2];
                        if (guac_client.onjoin) guac_client.onjoin(userID, username);
                        break;

                    case Guacamole.Client.Message.USER_LEFT:
                        userID = parameters[1];
                        username = parameters[2];
                        if (guac_client.onleave) guac_client.onleave(userID, username);
                        break;
                }
            }
        },

        name: function (parameters) {
            if (guac_client.onname) guac_client.onname(parameters[0]);
        },

        nest: function (parameters) {
            var parser = getParser(parseInt(parameters[0]));
            parser.receive(parameters[1]);
        },

        pipe: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var mimetype = parameters[1];
            var name = parameters[2];

            // Create stream
            if (guac_client.onpipe) {
                var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));
                guac_client.onpipe(stream, mimetype, name);
            }

            // Otherwise, unsupported
            else guac_client.sendAck(stream_index, 'Named pipes unsupported', 0x0100);
        },

        png: function (parameters) {
            var channelMask = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var x = parseInt(parameters[2]);
            var y = parseInt(parameters[3]);
            var data = parameters[4];

            display.setChannelMask(layer, channelMask);
            display.draw(layer, x, y, 'data:image/png;base64,' + data);
        },

        pop: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));

            display.pop(layer);
        },

        push: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));

            display.push(layer);
        },

        rect: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var x = parseInt(parameters[1]);
            var y = parseInt(parameters[2]);
            var w = parseInt(parameters[3]);
            var h = parseInt(parameters[4]);

            display.rect(layer, x, y, w, h);
        },

        required: function required(parameters) {
            if (guac_client.onrequired) guac_client.onrequired(parameters);
        },

        reset: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));

            display.reset(layer);
        },

        set: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var name = parameters[1];
            var value = parameters[2];

            // Call property handler if defined
            var handler = layerPropertyHandlers[name];
            if (handler) handler(layer, value);
        },

        shade: function (parameters) {
            var layer_index = parseInt(parameters[0]);
            var a = parseInt(parameters[1]);

            // Only valid for visible layers (not buffers)
            if (layer_index >= 0) {
                var layer = getLayer(layer_index);
                display.shade(layer, a);
            }
        },

        size: function (parameters) {
            var layer_index = parseInt(parameters[0]);
            var layer = getLayer(layer_index);
            var width = parseInt(parameters[1]);
            var height = parseInt(parameters[2]);

            display.resize(layer, width, height);
        },

        start: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var x = parseInt(parameters[1]);
            var y = parseInt(parameters[2]);

            display.moveTo(layer, x, y);
        },

        sync: function (parameters) {
            var timestamp = parseInt(parameters[0]);
            var frames = parameters[1] ? parseInt(parameters[1]) : 0;

            // Flush display, send sync when done
            display.flush(
                function displaySyncComplete() {
                    // Synchronize all audio players
                    for (var index in audioPlayers) {
                        var audioPlayer = audioPlayers[index];
                        if (audioPlayer) audioPlayer.sync();
                    }

                    // Send sync response to server
                    if (timestamp !== currentTimestamp) {
                        tunnel.sendMessage('sync', timestamp);
                        currentTimestamp = timestamp;
                    }
                },
                timestamp,
                frames
            );

            // If received first update, no longer waiting.
            if (currentState === Guacamole.Client.State.WAITING) setState(Guacamole.Client.State.CONNECTED);

            // Call sync handler if defined
            if (guac_client.onsync) guac_client.onsync(timestamp, frames);
        },

        transfer: function (parameters) {
            var srcL = getLayer(parseInt(parameters[0]));
            var srcX = parseInt(parameters[1]);
            var srcY = parseInt(parameters[2]);
            var srcWidth = parseInt(parameters[3]);
            var srcHeight = parseInt(parameters[4]);
            var function_index = parseInt(parameters[5]);
            var dstL = getLayer(parseInt(parameters[6]));
            var dstX = parseInt(parameters[7]);
            var dstY = parseInt(parameters[8]);

            /* SRC */
            if (function_index === 0x3) display.put(srcL, srcX, srcY, srcWidth, srcHeight, dstL, dstX, dstY);
            /* Anything else that isn't a NO-OP */ else if (function_index !== 0x5)
                display.transfer(srcL, srcX, srcY, srcWidth, srcHeight, dstL, dstX, dstY, Guacamole.Client.DefaultTransferFunction[function_index]);
        },

        transform: function (parameters) {
            var layer = getLayer(parseInt(parameters[0]));
            var a = parseFloat(parameters[1]);
            var b = parseFloat(parameters[2]);
            var c = parseFloat(parameters[3]);
            var d = parseFloat(parameters[4]);
            var e = parseFloat(parameters[5]);
            var f = parseFloat(parameters[6]);

            display.transform(layer, a, b, c, d, e, f);
        },

        undefine: function handleUndefine(parameters) {
            // Get object
            var objectIndex = parseInt(parameters[0]);
            var object = objects[objectIndex];

            // Signal end of object definition
            if (object && object.onundefine) object.onundefine();
        },

        video: function (parameters) {
            var stream_index = parseInt(parameters[0]);
            var layer = getLayer(parseInt(parameters[1]));
            var mimetype = parameters[2];

            // Create stream
            var stream = (streams[stream_index] = new Guacamole.InputStream(guac_client, stream_index));

            // Get player instance via callback
            var videoPlayer = null;
            if (guac_client.onvideo) videoPlayer = guac_client.onvideo(stream, layer, mimetype);

            // If unsuccessful, try to use a default implementation
            if (!videoPlayer) videoPlayer = Guacamole.VideoPlayer.getInstance(stream, layer, mimetype);

            // If we have successfully retrieved an video player, send success response
            if (videoPlayer) {
                videoPlayers[stream_index] = videoPlayer;
                guac_client.sendAck(stream_index, 'OK', 0x0000);
            }

            // Otherwise, mimetype must be unsupported
            else guac_client.sendAck(stream_index, 'BAD TYPE', 0x030f);
        },
    };

    /**
     * Sends a keep-alive ping to the Guacamole server, advising the server
     * that the client is still connected and responding. The lastSentKeepAlive
     * timestamp is automatically updated as a result of calling this function.
     *
     * @private
     */
    var sendKeepAlive = function sendKeepAlive() {
        tunnel.sendMessage('nop');
        lastSentKeepAlive = new Date().getTime();
    };

    /**
     * Schedules the next keep-alive ping based on the KEEP_ALIVE_FREQUENCY and
     * the time that the last ping was sent, if ever. If enough time has
     * elapsed that a ping should have already been sent, calling this function
     * will send that ping immediately.
     *
     * @private
     */
    var scheduleKeepAlive = function scheduleKeepAlive() {
        window.clearTimeout(keepAliveTimeout);

        var currentTime = new Date().getTime();
        var keepAliveDelay = Math.max(lastSentKeepAlive + KEEP_ALIVE_FREQUENCY - currentTime, 0);

        // Ping server regularly to keep connection alive, but send the ping
        // immediately if enough time has elapsed that it should have already
        // been sent
        if (keepAliveDelay > 0) keepAliveTimeout = window.setTimeout(sendKeepAlive, keepAliveDelay);
        else sendKeepAlive();
    };

    /**
     * Stops sending any further keep-alive pings. If a keep-alive ping was
     * scheduled to be sent, that ping is cancelled.
     *
     * @private
     */
    var stopKeepAlive = function stopKeepAlive() {
        window.clearTimeout(keepAliveTimeout);
    };

    tunnel.oninstruction = function (opcode, parameters) {
        var handler = instructionHandlers[opcode];
        if (handler) handler(parameters);

        // Leverage network activity to ensure the next keep-alive ping is
        // sent, even if the browser is currently throttling timers
        scheduleKeepAlive();
    };

    /**
     * Sends a disconnect instruction to the server and closes the tunnel.
     */
    this.disconnect = function () {
        // Only attempt disconnection not disconnected.
        if (currentState != Guacamole.Client.State.DISCONNECTED && currentState != Guacamole.Client.State.DISCONNECTING) {
            setState(Guacamole.Client.State.DISCONNECTING);

            // Stop sending keep-alive messages
            stopKeepAlive();

            // Send disconnect message and disconnect
            tunnel.sendMessage('disconnect');
            tunnel.disconnect();
            setState(Guacamole.Client.State.DISCONNECTED);
        }
    };

    /**
     * Connects the underlying tunnel of this Guacamole.Client, passing the
     * given arbitrary data to the tunnel during the connection process.
     *
     * @param {string} data
     *     Arbitrary connection data to be sent to the underlying tunnel during
     *     the connection process.
     *
     * @throws {!Guacamole.Status}
     *     If an error occurs during connection.
     */
    this.connect = function (data) {
        setState(Guacamole.Client.State.CONNECTING);

        try {
            tunnel.connect(data);
        } catch (status) {
            setState(Guacamole.Client.State.IDLE);
            throw status;
        }

        // Regularly send keep-alive ping to ensure the server knows we're
        // still here, even if not active
        scheduleKeepAlive();

        setState(Guacamole.Client.State.WAITING);
    };
};

/**
 * All possible Guacamole Client states.
 *
 * @type {!Object.<string, number>}
 */
Guacamole.Client.State = {
    /**
     * The client is idle, with no active connection.
     *
     * @type number
     */
    IDLE: 0,

    /**
     * The client is in the process of establishing a connection.
     *
     * @type {!number}
     */
    CONNECTING: 1,

    /**
     * The client is waiting on further information or a remote server to
     * establish the connection.
     *
     * @type {!number}
     */
    WAITING: 2,

    /**
     * The client is actively connected to a remote server.
     *
     * @type {!number}
     */
    CONNECTED: 3,

    /**
     * The client is in the process of disconnecting from the remote server.
     *
     * @type {!number}
     */
    DISCONNECTING: 4,

    /**
     * The client has completed the connection and is no longer connected.
     *
     * @type {!number}
     */
    DISCONNECTED: 5,
};

/**
 * Map of all Guacamole binary raster operations to transfer functions.
 *
 * @private
 * @type {!Object.<number, function>}
 */
Guacamole.Client.DefaultTransferFunction = {
    /* BLACK */
    0x0: function (src, dst) {
        dst.red = dst.green = dst.blue = 0x00;
    },

    /* WHITE */
    0xf: function (src, dst) {
        dst.red = dst.green = dst.blue = 0xff;
    },

    /* SRC */
    0x3: function (src, dst) {
        dst.red = src.red;
        dst.green = src.green;
        dst.blue = src.blue;
        dst.alpha = src.alpha;
    },

    /* DEST (no-op) */
    0x5: function (src, dst) {
        // Do nothing
    },

    /* Invert SRC */
    0xc: function (src, dst) {
        dst.red = 0xff & ~src.red;
        dst.green = 0xff & ~src.green;
        dst.blue = 0xff & ~src.blue;
        dst.alpha = src.alpha;
    },

    /* Invert DEST */
    0xa: function (src, dst) {
        dst.red = 0xff & ~dst.red;
        dst.green = 0xff & ~dst.green;
        dst.blue = 0xff & ~dst.blue;
    },

    /* AND */
    0x1: function (src, dst) {
        dst.red = src.red & dst.red;
        dst.green = src.green & dst.green;
        dst.blue = src.blue & dst.blue;
    },

    /* NAND */
    0xe: function (src, dst) {
        dst.red = 0xff & ~(src.red & dst.red);
        dst.green = 0xff & ~(src.green & dst.green);
        dst.blue = 0xff & ~(src.blue & dst.blue);
    },

    /* OR */
    0x7: function (src, dst) {
        dst.red = src.red | dst.red;
        dst.green = src.green | dst.green;
        dst.blue = src.blue | dst.blue;
    },

    /* NOR */
    0x8: function (src, dst) {
        dst.red = 0xff & ~(src.red | dst.red);
        dst.green = 0xff & ~(src.green | dst.green);
        dst.blue = 0xff & ~(src.blue | dst.blue);
    },

    /* XOR */
    0x6: function (src, dst) {
        dst.red = src.red ^ dst.red;
        dst.green = src.green ^ dst.green;
        dst.blue = src.blue ^ dst.blue;
    },

    /* XNOR */
    0x9: function (src, dst) {
        dst.red = 0xff & ~(src.red ^ dst.red);
        dst.green = 0xff & ~(src.green ^ dst.green);
        dst.blue = 0xff & ~(src.blue ^ dst.blue);
    },

    /* AND inverted source */
    0x4: function (src, dst) {
        dst.red = 0xff & (~src.red & dst.red);
        dst.green = 0xff & (~src.green & dst.green);
        dst.blue = 0xff & (~src.blue & dst.blue);
    },

    /* OR inverted source */
    0xd: function (src, dst) {
        dst.red = 0xff & (~src.red | dst.red);
        dst.green = 0xff & (~src.green | dst.green);
        dst.blue = 0xff & (~src.blue | dst.blue);
    },

    /* AND inverted destination */
    0x2: function (src, dst) {
        dst.red = 0xff & (src.red & ~dst.red);
        dst.green = 0xff & (src.green & ~dst.green);
        dst.blue = 0xff & (src.blue & ~dst.blue);
    },

    /* OR inverted destination */
    0xb: function (src, dst) {
        dst.red = 0xff & (src.red | ~dst.red);
        dst.green = 0xff & (src.green | ~dst.green);
        dst.blue = 0xff & (src.blue | ~dst.blue);
    },
};

/**
 * A list of possible messages that can be sent by the server for processing
 * by the client.
 *
 * @type {!Object.<string, number>}
 */
Guacamole.Client.Message = {
    /**
     * A client message that indicates that a user has joined an existing
     * connection. This message expects a single additional argument - the
     * name of the user who has joined the connection.
     *
     * @type {!number}
     */
    USER_JOINED: 0x0001,

    /**
     * A client message that indicates that a user has left an existing
     * connection. This message expects a single additional argument - the
     * name of the user who has left the connection.
     *
     * @type {!number}
     */
    USER_LEFT: 0x0002,
};

/**
 * A reader which automatically handles the given input stream, returning
 * received blobs as a single data URI built over the course of the stream.
 * Note that this object will overwrite any installed event handlers on the
 * given Guacamole.InputStream.
 *
 * @constructor
 * @param {!Guacamole.InputStream} stream
 *     The stream that data will be read from.
 *
 * @param {!string} mimetype
 *     The mimetype of the data being received.
 */
Guacamole.DataURIReader = function (stream, mimetype) {
    /**
     * Reference to this Guacamole.DataURIReader.
     *
     * @private
     * @type {!Guacamole.DataURIReader}
     */
    var guac_reader = this;

    /**
     * Current data URI.
     *
     * @private
     * @type {!string}
     */
    var uri = 'data:' + mimetype + ';base64,';

    // Receive blobs as array buffers
    stream.onblob = function dataURIReaderBlob(data) {
        // Currently assuming data will ALWAYS be safe to simply append. This
        // will not be true if the received base64 data encodes a number of
        // bytes that isn't a multiple of three (as base64 expands in a ratio
        // of exactly 3:4).
        uri += data;
    };

    // Simply call onend when end received
    stream.onend = function dataURIReaderEnd() {
        if (guac_reader.onend) guac_reader.onend();
    };

    /**
     * Returns the data URI of all data received through the underlying stream
     * thus far.
     *
     * @returns {!string}
     *     The data URI of all data received through the underlying stream thus
     *     far.
     */
    this.getURI = function getURI() {
        return uri;
    };

    /**
     * Fired once this stream is finished and no further data will be written.
     *
     * @event
     */
    this.onend = null;
};

/**
 * The Guacamole display. The display does not deal with the Guacamole
 * protocol, and instead implements a set of graphical operations which
 * embody the set of operations present in the protocol. The order operations
 * are executed is guaranteed to be in the same order as their corresponding
 * functions are called.
 *
 * @constructor
 */
Guacamole.Display = function () {
    /**
     * Reference to this Guacamole.Display.
     * @private
     */
    var guac_display = this;

    var displayWidth = 0;
    var displayHeight = 0;
    var displayScale = 1;

    // Create display
    var display = document.createElement('div');
    display.style.position = 'relative';
    display.style.width = displayWidth + 'px';
    display.style.height = displayHeight + 'px';

    // Ensure transformations on display originate at 0,0
    display.style.transformOrigin =
        display.style.webkitTransformOrigin =
        display.style.MozTransformOrigin =
        display.style.OTransformOrigin =
        display.style.msTransformOrigin =
            '0 0';

    // Create default layer
    var default_layer = new Guacamole.Display.VisibleLayer(displayWidth, displayHeight);

    // Create cursor layer
    var cursor = new Guacamole.Display.VisibleLayer(0, 0);
    cursor.setChannelMask(Guacamole.Layer.SRC);

    // Add default layer and cursor to display
    display.appendChild(default_layer.getElement());
    display.appendChild(cursor.getElement());

    // Create bounding div
    var bounds = document.createElement('div');
    bounds.style.position = 'relative';
    bounds.style.width = displayWidth * displayScale + 'px';
    bounds.style.height = displayHeight * displayScale + 'px';

    // Add display to bounds
    bounds.appendChild(display);

    /**
     * The X coordinate of the hotspot of the mouse cursor. The hotspot is
     * the relative location within the image of the mouse cursor at which
     * each click occurs.
     *
     * @type {!number}
     */
    this.cursorHotspotX = 0;

    /**
     * The Y coordinate of the hotspot of the mouse cursor. The hotspot is
     * the relative location within the image of the mouse cursor at which
     * each click occurs.
     *
     * @type {!number}
     */
    this.cursorHotspotY = 0;

    /**
     * The current X coordinate of the local mouse cursor. This is not
     * necessarily the location of the actual mouse - it refers only to
     * the location of the cursor image within the Guacamole display, as
     * last set by moveCursor().
     *
     * @type {!number}
     */
    this.cursorX = 0;

    /**
     * The current X coordinate of the local mouse cursor. This is not
     * necessarily the location of the actual mouse - it refers only to
     * the location of the cursor image within the Guacamole display, as
     * last set by moveCursor().
     *
     * @type {!number}
     */
    this.cursorY = 0;

    /**
     * The number of milliseconds over which display rendering statistics
     * should be gathered, dispatching {@link #onstatistics} events as those
     * statistics are available. If set to zero, no statistics will be
     * gathered.
     *
     * @default 0
     * @type {!number}
     */
    this.statisticWindow = 0;

    /**
     * Fired when the default layer (and thus the entire Guacamole display)
     * is resized.
     *
     * @event
     * @param {!number} width
     *     The new width of the Guacamole display.
     *
     * @param {!number} height
     *     The new height of the Guacamole display.
     */
    this.onresize = null;

    /**
     * Fired whenever the local cursor image is changed. This can be used to
     * implement special handling of the client-side cursor, or to override
     * the default use of a software cursor layer.
     *
     * @event
     * @param {!HTMLCanvasElement} canvas
     *     The cursor image.
     *
     * @param {!number} x
     *     The X-coordinate of the cursor hotspot.
     *
     * @param {!number} y
     *     The Y-coordinate of the cursor hotspot.
     */
    this.oncursor = null;

    /**
     * Fired whenever performance statistics are available for recently-
     * rendered frames. This event will fire only if {@link #statisticWindow}
     * is non-zero.
     *
     * @event
     * @param {!Guacamole.Display.Statistics} stats
     *     An object containing general rendering performance statistics for
     *     the remote desktop, Guacamole server, and Guacamole client.
     */
    this.onstatistics = null;

    /**
     * The queue of all pending Tasks. Tasks will be run in order, with new
     * tasks added at the end of the queue and old tasks removed from the
     * front of the queue (FIFO). These tasks will eventually be grouped
     * into a Frame.
     *
     * @private
     * @type {!Task[]}
     */
    var tasks = [];

    /**
     * The queue of all frames. Each frame is a pairing of an array of tasks
     * and a callback which must be called when the frame is rendered.
     *
     * @private
     * @type {!Frame[]}
     */
    var frames = [];

    /**
     * The ID of the animation frame request returned by the last call to
     * requestAnimationFrame(). This value will only be set if the browser
     * supports requestAnimationFrame(), if a frame render is currently
     * pending, and if the current browser tab is currently focused (likely to
     * handle requests for animation frames). In all other cases, this will be
     * null.
     *
     * @private
     * @type {number}
     */
    var inProgressFrame = null;

    /**
     * Flushes all pending frames synchronously. This function will block until
     * all pending frames have rendered. If a frame is currently blocked by an
     * asynchronous operation like an image load, this function will return
     * after reaching that operation and the flush operation will
     * automamtically resume after that operation completes.
     *
     * @private
     */
    var syncFlush = function syncFlush() {
        var localTimestamp = 0;
        var remoteTimestamp = 0;

        var renderedLogicalFrames = 0;
        var rendered_frames = 0;

        // Draw all pending frames, if ready
        while (rendered_frames < frames.length) {
            var frame = frames[rendered_frames];
            if (!frame.isReady()) break;

            frame.flush();

            localTimestamp = frame.localTimestamp;
            remoteTimestamp = frame.remoteTimestamp;
            renderedLogicalFrames += frame.logicalFrames;
            rendered_frames++;
        }

        // Remove rendered frames from array
        frames.splice(0, rendered_frames);

        if (rendered_frames) notifyFlushed(localTimestamp, remoteTimestamp, renderedLogicalFrames);
    };

    /**
     * Flushes all pending frames asynchronously. This function returns
     * immediately, relying on requestAnimationFrame() to dictate when each
     * frame should be flushed.
     *
     * @private
     */
    var asyncFlush = function asyncFlush() {
        var continueFlush = function continueFlush() {
            // We're no longer waiting to render a frame
            inProgressFrame = null;

            // Nothing to do if there are no frames remaining
            if (!frames.length) return;

            // Flush the next frame only if it is ready (not awaiting
            // completion of some asynchronous operation like an image load)
            if (frames[0].isReady()) {
                var frame = frames.shift();
                frame.flush();
                notifyFlushed(frame.localTimestamp, frame.remoteTimestamp, frame.logicalFrames);
            }

            // Request yet another animation frame if frames remain to be
            // flushed
            if (frames.length) inProgressFrame = window.requestAnimationFrame(continueFlush);
        };

        // Begin flushing frames if not already waiting to render a frame
        if (!inProgressFrame) inProgressFrame = window.requestAnimationFrame(continueFlush);
    };

    /**
     * Recently-gathered display render statistics, as made available by calls
     * to notifyFlushed(). The contents of this array will be trimmed to
     * contain only up to {@link #statisticWindow} milliseconds of statistics.
     *
     * @private
     * @type {Guacamole.Display.Statistics[]}
     */
    var statistics = [];

    /**
     * Notifies that one or more frames have been successfully rendered
     * (flushed) to the display.
     *
     * @private
     * @param {!number} localTimestamp
     *     The local timestamp of the point in time at which the most recent,
     *     flushed frame was received by the display, in milliseconds since the
     *     Unix Epoch.
     *
     * @param {!number} remoteTimestamp
     *     The remote timestamp of sync instruction associated with the most
     *     recent, flushed frame received by the display. This timestamp is in
     *     milliseconds, but is arbitrary, having meaning only relative to
     *     other timestamps in the same connection.
     *
     * @param {!number} logicalFrames
     *     The number of remote desktop frames that were flushed.
     */
    var notifyFlushed = function notifyFlushed(localTimestamp, remoteTimestamp, logicalFrames) {
        // Ignore if statistics are not being gathered
        if (!guac_display.statisticWindow) return;

        var current = new Date().getTime();

        // Find the first statistic that is still within the configured time
        // window
        for (var first = 0; first < statistics.length; first++) {
            if (current - statistics[first].timestamp <= guac_display.statisticWindow) break;
        }

        // Remove all statistics except those within the time window
        statistics.splice(0, first - 1);

        // Record statistics for latest frame
        statistics.push({
            localTimestamp: localTimestamp,
            remoteTimestamp: remoteTimestamp,
            timestamp: current,
            frames: logicalFrames,
        });

        // Determine the actual time interval of the available statistics (this
        // will not perfectly match the configured interval, which is an upper
        // bound)
        var statDuration = (statistics[statistics.length - 1].timestamp - statistics[0].timestamp) / 1000;

        // Determine the amount of time that elapsed remotely (within the
        // remote desktop)
        var remoteDuration = (statistics[statistics.length - 1].remoteTimestamp - statistics[0].remoteTimestamp) / 1000;

        // Calculate the number of frames that have been rendered locally
        // within the configured time interval
        var localFrames = statistics.length;

        // Calculate the number of frames actually received from the remote
        // desktop by the Guacamole server
        var remoteFrames = statistics.reduce(function sumFrames(prev, stat) {
            return prev + stat.frames;
        }, 0);

        // Calculate the number of frames that the Guacamole server had to
        // drop or combine with other frames
        var drops = statistics.reduce(function sumDrops(prev, stat) {
            return prev + Math.max(0, stat.frames - 1);
        }, 0);

        // Produce lag and FPS statistics from above raw measurements
        var stats = new Guacamole.Display.Statistics({
            processingLag: current - localTimestamp,
            desktopFps: remoteDuration && remoteFrames ? remoteFrames / remoteDuration : null,
            clientFps: statDuration ? localFrames / statDuration : null,
            serverFps: remoteDuration ? localFrames / remoteDuration : null,
            dropRate: remoteDuration ? drops / remoteDuration : null,
        });

        // Notify of availability of new statistics
        if (guac_display.onstatistics) guac_display.onstatistics(stats);
    };

    // Switch from asynchronous frame handling to synchronous frame handling if
    // requestAnimationFrame() is unlikely to be usable (browsers may not
    // invoke the animation frame callback if the relevant tab is not focused)
    window.addEventListener(
        'blur',
        function switchToSyncFlush() {
            if (inProgressFrame && !document.hasFocus()) {
                // Cancel pending asynchronous processing of frame ...
                window.cancelAnimationFrame(inProgressFrame);
                inProgressFrame = null;

                // ... and instead process it synchronously
                syncFlush();
            }
        },
        true
    );

    /**
     * Flushes all pending frames.
     * @private
     */
    function __flush_frames() {
        if (window.requestAnimationFrame && document.hasFocus()) asyncFlush();
        else syncFlush();
    }

    /**
     * An ordered list of tasks which must be executed atomically. Once
     * executed, an associated (and optional) callback will be called.
     *
     * @private
     * @constructor
     * @param {function} [callback]
     *     The function to call when this frame is rendered.
     *
     * @param {!Task[]} tasks
     *     The set of tasks which must be executed to render this frame.
     *
     * @param {number} [timestamp]
     *     The remote timestamp of sync instruction associated with this frame.
     *     This timestamp is in milliseconds, but is arbitrary, having meaning
     *     only relative to other remote timestamps in the same connection. If
     *     omitted, a compatible but local timestamp will be used instead.
     *
     * @param {number} [logicalFrames=0]
     *     The number of remote desktop frames that were combined to produce
     *     this frame, or zero if this value is unknown or inapplicable.
     */
    var Frame = function Frame(callback, tasks, timestamp, logicalFrames) {
        /**
         * The local timestamp of the point in time at which this frame was
         * received by the display, in milliseconds since the Unix Epoch.
         *
         * @type {!number}
         */
        this.localTimestamp = new Date().getTime();

        /**
         * The remote timestamp of sync instruction associated with this frame.
         * This timestamp is in milliseconds, but is arbitrary, having meaning
         * only relative to other remote timestamps in the same connection.
         *
         * @type {!number}
         */
        this.remoteTimestamp = timestamp || this.localTimestamp;

        /**
         * The number of remote desktop frames that were combined to produce
         * this frame. If unknown or not applicable, this will be zero.
         *
         * @type {!number}
         */
        this.logicalFrames = logicalFrames || 0;

        /**
         * Cancels rendering of this frame and all associated tasks. The
         * callback provided at construction time, if any, is not invoked.
         */
        this.cancel = function cancel() {
            callback = null;

            tasks.forEach(function cancelTask(task) {
                task.cancel();
            });

            tasks = [];
        };

        /**
         * Returns whether this frame is ready to be rendered. This function
         * returns true if and only if ALL underlying tasks are unblocked.
         *
         * @returns {!boolean}
         *     true if all underlying tasks are unblocked, false otherwise.
         */
        this.isReady = function () {
            // Search for blocked tasks
            for (var i = 0; i < tasks.length; i++) {
                if (tasks[i].blocked) return false;
            }

            // If no blocked tasks, the frame is ready
            return true;
        };

        /**
         * Renders this frame, calling the associated callback, if any, after
         * the frame is complete. This function MUST only be called when no
         * blocked tasks exist. Calling this function with blocked tasks
         * will result in undefined behavior.
         */
        this.flush = function () {
            // Draw all pending tasks.
            for (var i = 0; i < tasks.length; i++) tasks[i].execute();

            // Call callback
            if (callback) callback();
        };
    };

    /**
     * A container for an task handler. Each operation which must be ordered
     * is associated with a Task that goes into a task queue. Tasks in this
     * queue are executed in order once their handlers are set, while Tasks
     * without handlers block themselves and any following Tasks from running.
     *
     * @constructor
     * @private
     * @param {function} [taskHandler]
     *     The function to call when this task runs, if any.
     *
     * @param {boolean} [blocked]
     *     Whether this task should start blocked.
     */
    function Task(taskHandler, blocked) {
        /**
         * Reference to this Task.
         *
         * @private
         * @type {!Guacamole.Display.Task}
         */
        var task = this;

        /**
         * Whether this Task is blocked.
         *
         * @type {boolean}
         */
        this.blocked = blocked;

        /**
         * Cancels this task such that it will not run. The task handler
         * provided at construction time, if any, is not invoked. Calling
         * execute() after calling this function has no effect.
         */
        this.cancel = function cancel() {
            task.blocked = false;
            taskHandler = null;
        };

        /**
         * Unblocks this Task, allowing it to run.
         */
        this.unblock = function () {
            if (task.blocked) {
                task.blocked = false;
                __flush_frames();
            }
        };

        /**
         * Calls the handler associated with this task IMMEDIATELY. This
         * function does not track whether this task is marked as blocked.
         * Enforcing the blocked status of tasks is up to the caller.
         */
        this.execute = function () {
            if (taskHandler) taskHandler();
        };
    }

    /**
     * Schedules a task for future execution. The given handler will execute
     * immediately after all previous tasks upon frame flush, unless this
     * task is blocked. If any tasks is blocked, the entire frame will not
     * render (and no tasks within will execute) until all tasks are unblocked.
     *
     * @private
     * @param {function} [handler]
     *     The function to call when possible, if any.
     *
     * @param {boolean} [blocked]
     *     Whether the task should start blocked.
     *
     * @returns {!Task}
     *     The Task created and added to the queue for future running.
     */
    function scheduleTask(handler, blocked) {
        var task = new Task(handler, blocked);
        tasks.push(task);
        return task;
    }

    /**
     * Returns the element which contains the Guacamole display.
     *
     * @return {!Element}
     *     The element containing the Guacamole display.
     */
    this.getElement = function () {
        return bounds;
    };

    /**
     * Returns the width of this display.
     *
     * @return {!number}
     *     The width of this display;
     */
    this.getWidth = function () {
        return displayWidth;
    };

    /**
     * Returns the height of this display.
     *
     * @return {!number}
     *     The height of this display;
     */
    this.getHeight = function () {
        return displayHeight;
    };

    /**
     * Returns the default layer of this display. Each Guacamole display always
     * has at least one layer. Other layers can optionally be created within
     * this layer, but the default layer cannot be removed and is the absolute
     * ancestor of all other layers.
     *
     * @return {!Guacamole.Display.VisibleLayer}
     *     The default layer.
     */
    this.getDefaultLayer = function () {
        return default_layer;
    };

    /**
     * Returns the cursor layer of this display. Each Guacamole display contains
     * a layer for the image of the mouse cursor. This layer is a special case
     * and exists above all other layers, similar to the hardware mouse cursor.
     *
     * @return {!Guacamole.Display.VisibleLayer}
     *     The cursor layer.
     */
    this.getCursorLayer = function () {
        return cursor;
    };

    /**
     * Creates a new layer. The new layer will be a direct child of the default
     * layer, but can be moved to be a child of any other layer. Layers returned
     * by this function are visible.
     *
     * @return {!Guacamole.Display.VisibleLayer}
     *     The newly-created layer.
     */
    this.createLayer = function () {
        var layer = new Guacamole.Display.VisibleLayer(displayWidth, displayHeight);
        layer.move(default_layer, 0, 0, 0);
        return layer;
    };

    /**
     * Creates a new buffer. Buffers are invisible, off-screen surfaces. They
     * are implemented in the same manner as layers, but do not provide the
     * same nesting semantics.
     *
     * @return {!Guacamole.Layer}
     *     The newly-created buffer.
     */
    this.createBuffer = function () {
        var buffer = new Guacamole.Layer(0, 0);
        buffer.autosize = 1;
        return buffer;
    };

    /**
     * Flush all pending draw tasks, if possible, as a new frame. If the entire
     * frame is not ready, the flush will wait until all required tasks are
     * unblocked.
     *
     * @param {function} [callback]
     *     The function to call when this frame is flushed. This may happen
     *     immediately, or later when blocked tasks become unblocked.
     *
     * @param {number} timestamp
     *     The remote timestamp of sync instruction associated with this frame.
     *     This timestamp is in milliseconds, but is arbitrary, having meaning
     *     only relative to other remote timestamps in the same connection.
     *
     * @param {number} logicalFrames
     *     The number of remote desktop frames that were combined to produce
     *     this frame.
     */
    this.flush = function (callback, timestamp, logicalFrames) {
        // Add frame, reset tasks
        frames.push(new Frame(callback, tasks, timestamp, logicalFrames));
        tasks = [];

        // Attempt flush
        __flush_frames();
    };

    /**
     * Cancels rendering of all pending frames and associated rendering
     * operations. The callbacks provided to outstanding past calls to flush(),
     * if any, are not invoked.
     */
    this.cancel = function cancel() {
        frames.forEach(function cancelFrame(frame) {
            frame.cancel();
        });

        frames = [];

        tasks.forEach(function cancelTask(task) {
            task.cancel();
        });

        tasks = [];
    };

    /**
     * Sets the hotspot and image of the mouse cursor displayed within the
     * Guacamole display.
     *
     * @param {!number} hotspotX
     *     The X coordinate of the cursor hotspot.
     *
     * @param {!number} hotspotY
     *     The Y coordinate of the cursor hotspot.
     *
     * @param {!Guacamole.Layer} layer
     *     The source layer containing the data which should be used as the
     *     mouse cursor image.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source layer's coordinate
     *     space to copy data from.
     */
    this.setCursor = function (hotspotX, hotspotY, layer, srcx, srcy, srcw, srch) {
        scheduleTask(function __display_set_cursor() {
            // Set hotspot
            guac_display.cursorHotspotX = hotspotX;
            guac_display.cursorHotspotY = hotspotY;

            // Reset cursor size
            cursor.resize(srcw, srch);

            // Draw cursor to cursor layer
            cursor.copy(layer, srcx, srcy, srcw, srch, 0, 0);
            guac_display.moveCursor(guac_display.cursorX, guac_display.cursorY);

            // Fire cursor change event
            if (guac_display.oncursor) guac_display.oncursor(cursor.toCanvas(), hotspotX, hotspotY);
        });
    };

    /**
     * Sets whether the software-rendered cursor is shown. This cursor differs
     * from the hardware cursor in that it is built into the Guacamole.Display,
     * and relies on its own Guacamole layer to render.
     *
     * @param {boolean} [shown=true]
     *     Whether to show the software cursor.
     */
    this.showCursor = function (shown) {
        var element = cursor.getElement();
        var parent = element.parentNode;

        // Remove from DOM if hidden
        if (shown === false) {
            if (parent) parent.removeChild(element);
        }

        // Otherwise, ensure cursor is child of display
        else if (parent !== display) display.appendChild(element);
    };

    /**
     * Sets the location of the local cursor to the given coordinates. For the
     * sake of responsiveness, this function performs its action immediately.
     * Cursor motion is not maintained within atomic frames.
     *
     * @param {!number} x
     *     The X coordinate to move the cursor to.
     *
     * @param {!number} y
     *     The Y coordinate to move the cursor to.
     */
    this.moveCursor = function (x, y) {
        // Move cursor layer
        cursor.translate(x - guac_display.cursorHotspotX, y - guac_display.cursorHotspotY);

        // Update stored position
        guac_display.cursorX = x;
        guac_display.cursorY = y;
    };

    /**
     * Changes the size of the given Layer to the given width and height.
     * Resizing is only attempted if the new size provided is actually different
     * from the current size.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to resize.
     *
     * @param {!number} width
     *     The new width.
     *
     * @param {!number} height
     *     The new height.
     */
    this.resize = function (layer, width, height) {
        scheduleTask(function __display_resize() {
            layer.resize(width, height);

            // Resize display if default layer is resized
            if (layer === default_layer) {
                // Update (set) display size
                displayWidth = width;
                displayHeight = height;
                display.style.width = displayWidth + 'px';
                display.style.height = displayHeight + 'px';

                // Update bounds size
                bounds.style.width = displayWidth * displayScale + 'px';
                bounds.style.height = displayHeight * displayScale + 'px';

                // Notify of resize
                if (guac_display.onresize) guac_display.onresize(width, height);
            }
        });
    };

    /**
     * Draws the specified image at the given coordinates. The image specified
     * must already be loaded.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!CanvasImageSource} image
     *     The image to draw. Note that this not a URL.
     */
    this.drawImage = function (layer, x, y, image) {
        scheduleTask(function __display_drawImage() {
            layer.drawImage(x, y, image);
        });
    };

    /**
     * Draws the image contained within the specified Blob at the given
     * coordinates. The Blob specified must already be populated with image
     * data.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!Blob} blob
     *     The Blob containing the image data to draw.
     */
    this.drawBlob = function (layer, x, y, blob) {
        var task;

        // Prefer createImageBitmap() over blob URLs if available
        if (window.createImageBitmap) {
            var bitmap;

            // Draw image once loaded
            task = scheduleTask(function drawImageBitmap() {
                layer.drawImage(x, y, bitmap);
            }, true);

            // Load image from provided blob
            window.createImageBitmap(blob).then(function bitmapLoaded(decoded) {
                bitmap = decoded;
                task.unblock();
            });
        }

        // Use blob URLs and the Image object if createImageBitmap() is
        // unavailable
        else {
            // Create URL for blob
            var url = URL.createObjectURL(blob);

            // Draw and free blob URL when ready
            task = scheduleTask(function __display_drawBlob() {
                // Draw the image only if it loaded without errors
                if (image.width && image.height) layer.drawImage(x, y, image);

                // Blob URL no longer needed
                URL.revokeObjectURL(url);
            }, true);

            // Load image from URL
            var image = new Image();
            image.onload = task.unblock;
            image.onerror = task.unblock;
            image.src = url;
        }
    };

    /**
     * Draws the image within the given stream at the given coordinates. The
     * image will be loaded automatically, and this and any future operations
     * will wait for the image to finish loading. This function will
     * automatically choose an appropriate method for reading and decoding the
     * given image stream, and should be preferred for received streams except
     * where manual decoding of the stream is unavoidable.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!Guacamole.InputStream} stream
     *     The stream along which image data will be received.
     *
     * @param {!string} mimetype
     *     The mimetype of the image within the stream.
     */
    this.drawStream = function drawStream(layer, x, y, stream, mimetype) {
        // If createImageBitmap() is available, load the image as a blob so
        // that function can be used
        if (window.createImageBitmap) {
            var reader = new Guacamole.BlobReader(stream, mimetype);
            reader.onend = function drawImageBlob() {
                guac_display.drawBlob(layer, x, y, reader.getBlob());
            };
        }

        // Lacking createImageBitmap(), fall back to data URIs and the Image
        // object
        else {
            let reader = new Guacamole.DataURIReader(stream, mimetype);
            reader.onend = function drawImageDataURI() {
                guac_display.draw(layer, x, y, reader.getURI());
            };
        }
    };

    /**
     * Draws the image at the specified URL at the given coordinates. The image
     * will be loaded automatically, and this and any future operations will
     * wait for the image to finish loading.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!string} url
     *     The URL of the image to draw.
     */
    this.draw = function (layer, x, y, url) {
        var task = scheduleTask(function __display_draw() {
            // Draw the image only if it loaded without errors
            if (image.width && image.height) layer.drawImage(x, y, image);
        }, true);

        var image = new Image();
        image.onload = task.unblock;
        image.onerror = task.unblock;
        image.src = url;
    };

    /**
     * Plays the video at the specified URL within this layer. The video
     * will be loaded automatically, and this and any future operations will
     * wait for the video to finish loading. Future operations will not be
     * executed until the video finishes playing.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!string} mimetype
     *     The mimetype of the video to play.
     *
     * @param {!number} duration
     *     The duration of the video in milliseconds.
     *
     * @param {!string} url
     *     The URL of the video to play.
     */
    this.play = function (layer, mimetype, duration, url) {
        // Start loading the video
        var video = document.createElement('video');
        video.type = mimetype;
        video.src = url;

        // Start copying frames when playing
        video.addEventListener(
            'play',
            function () {
                function render_callback() {
                    layer.drawImage(0, 0, video);
                    if (!video.ended) window.setTimeout(render_callback, 20);
                }

                render_callback();
            },
            false
        );

        scheduleTask(video.play);
    };

    /**
     * Transfer a rectangle of image data from one Layer to this Layer using the
     * specified transfer function.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The Layer to copy image data from.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!Guacamole.Layer} dstLayer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!function} transferFunction
     *     The transfer function to use to transfer data from source to
     *     destination.
     */
    this.transfer = function (srcLayer, srcx, srcy, srcw, srch, dstLayer, x, y, transferFunction) {
        scheduleTask(function __display_transfer() {
            dstLayer.transfer(srcLayer, srcx, srcy, srcw, srch, x, y, transferFunction);
        });
    };

    /**
     * Put a rectangle of image data from one Layer to this Layer directly
     * without performing any alpha blending. Simply copy the data.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The Layer to copy image data from.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!Guacamole.Layer} dstLayer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     */
    this.put = function (srcLayer, srcx, srcy, srcw, srch, dstLayer, x, y) {
        scheduleTask(function __display_put() {
            dstLayer.put(srcLayer, srcx, srcy, srcw, srch, x, y);
        });
    };

    /**
     * Copy a rectangle of image data from one Layer to this Layer. This
     * operation will copy exactly the image data that will be drawn once all
     * operations of the source Layer that were pending at the time this
     * function was called are complete. This operation will not alter the
     * size of the source Layer even if its autosize property is set to true.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The Layer to copy image data from.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source Layer's coordinate space to copy data from.
     *
     * @param {!Guacamole.Layer} dstLayer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     */
    this.copy = function (srcLayer, srcx, srcy, srcw, srch, dstLayer, x, y) {
        scheduleTask(function __display_copy() {
            dstLayer.copy(srcLayer, srcx, srcy, srcw, srch, x, y);
        });
    };

    /**
     * Starts a new path at the specified point.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The X coordinate of the point to draw.
     *
     * @param {!number} y
     *     The Y coordinate of the point to draw.
     */
    this.moveTo = function (layer, x, y) {
        scheduleTask(function __display_moveTo() {
            layer.moveTo(x, y);
        });
    };

    /**
     * Add the specified line to the current path.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The X coordinate of the endpoint of the line to draw.
     *
     * @param {!number} y
     *     The Y coordinate of the endpoint of the line to draw.
     */
    this.lineTo = function (layer, x, y) {
        scheduleTask(function __display_lineTo() {
            layer.lineTo(x, y);
        });
    };

    /**
     * Add the specified arc to the current path.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The X coordinate of the center of the circle which will contain the
     *     arc.
     *
     * @param {!number} y
     *     The Y coordinate of the center of the circle which will contain the
     *     arc.
     *
     * @param {!number} radius
     *     The radius of the circle.
     *
     * @param {!number} startAngle
     *     The starting angle of the arc, in radians.
     *
     * @param {!number} endAngle
     *     The ending angle of the arc, in radians.
     *
     * @param {!boolean} negative
     *     Whether the arc should be drawn in order of decreasing angle.
     */
    this.arc = function (layer, x, y, radius, startAngle, endAngle, negative) {
        scheduleTask(function __display_arc() {
            layer.arc(x, y, radius, startAngle, endAngle, negative);
        });
    };

    /**
     * Starts a new path at the specified point.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} cp1x
     *     The X coordinate of the first control point.
     *
     * @param {!number} cp1y
     *     The Y coordinate of the first control point.
     *
     * @param {!number} cp2x
     *     The X coordinate of the second control point.
     *
     * @param {!number} cp2y
     *     The Y coordinate of the second control point.
     *
     * @param {!number} x
     *     The X coordinate of the endpoint of the curve.
     *
     * @param {!number} y
     *     The Y coordinate of the endpoint of the curve.
     */
    this.curveTo = function (layer, cp1x, cp1y, cp2x, cp2y, x, y) {
        scheduleTask(function __display_curveTo() {
            layer.curveTo(cp1x, cp1y, cp2x, cp2y, x, y);
        });
    };

    /**
     * Closes the current path by connecting the end point with the start
     * point (if any) with a straight line.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     */
    this.close = function (layer) {
        scheduleTask(function __display_close() {
            layer.close();
        });
    };

    /**
     * Add the specified rectangle to the current path.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} x
     *     The X coordinate of the upper-left corner of the rectangle to draw.
     *
     * @param {!number} y
     *     The Y coordinate of the upper-left corner of the rectangle to draw.
     *
     * @param {!number} w
     *     The width of the rectangle to draw.
     *
     * @param {!number} h
     *     The height of the rectangle to draw.
     */
    this.rect = function (layer, x, y, w, h) {
        scheduleTask(function __display_rect() {
            layer.rect(x, y, w, h);
        });
    };

    /**
     * Clip all future drawing operations by the current path. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as fillColor()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to affect.
     */
    this.clip = function (layer) {
        scheduleTask(function __display_clip() {
            layer.clip();
        });
    };

    /**
     * Stroke the current path with the specified color. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!string} cap
     *     The line cap style. Can be "round", "square", or "butt".
     *
     * @param {!string} join
     *     The line join style. Can be "round", "bevel", or "miter".
     *
     * @param {!number} thickness
     *     The line thickness in pixels.
     *
     * @param {!number} r
     *     The red component of the color to fill.
     *
     * @param {!number} g
     *     The green component of the color to fill.
     *
     * @param {!number} b
     *     The blue component of the color to fill.
     *
     * @param {!number} a
     *     The alpha component of the color to fill.
     */
    this.strokeColor = function (layer, cap, join, thickness, r, g, b, a) {
        scheduleTask(function __display_strokeColor() {
            layer.strokeColor(cap, join, thickness, r, g, b, a);
        });
    };

    /**
     * Fills the current path with the specified color. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!number} r
     *     The red component of the color to fill.
     *
     * @param {!number} g
     *     The green component of the color to fill.
     *
     * @param {!number} b
     *     The blue component of the color to fill.
     *
     * @param {!number} a
     *     The alpha component of the color to fill.
     */
    this.fillColor = function (layer, r, g, b, a) {
        scheduleTask(function __display_fillColor() {
            layer.fillColor(r, g, b, a);
        });
    };

    /**
     * Stroke the current path with the image within the specified layer. The
     * image data will be tiled infinitely within the stroke. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!string} cap
     *     The line cap style. Can be "round", "square", or "butt".
     *
     * @param {!string} join
     *     The line join style. Can be "round", "bevel", or "miter".
     *
     * @param {!number} thickness
     *     The line thickness in pixels.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The layer to use as a repeating pattern within the stroke.
     */
    this.strokeLayer = function (layer, cap, join, thickness, srcLayer) {
        scheduleTask(function __display_strokeLayer() {
            layer.strokeLayer(cap, join, thickness, srcLayer);
        });
    };

    /**
     * Fills the current path with the image within the specified layer. The
     * image data will be tiled infinitely within the stroke. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The layer to use as a repeating pattern within the fill.
     */
    this.fillLayer = function (layer, srcLayer) {
        scheduleTask(function __display_fillLayer() {
            layer.fillLayer(srcLayer);
        });
    };

    /**
     * Push current layer state onto stack.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     */
    this.push = function (layer) {
        scheduleTask(function __display_push() {
            layer.push();
        });
    };

    /**
     * Pop layer state off stack.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     */
    this.pop = function (layer) {
        scheduleTask(function __display_pop() {
            layer.pop();
        });
    };

    /**
     * Reset the layer, clearing the stack, the current path, and any transform
     * matrix.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to draw upon.
     */
    this.reset = function (layer) {
        scheduleTask(function __display_reset() {
            layer.reset();
        });
    };

    /**
     * Sets the given affine transform (defined with six values from the
     * transform's matrix).
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to modify.
     *
     * @param {!number} a
     *     The first value in the affine transform's matrix.
     *
     * @param {!number} b
     *     The second value in the affine transform's matrix.
     *
     * @param {!number} c
     *     The third value in the affine transform's matrix.
     *
     * @param {!number} d
     *     The fourth value in the affine transform's matrix.
     *
     * @param {!number} e
     *     The fifth value in the affine transform's matrix.
     *
     * @param {!number} f
     *     The sixth value in the affine transform's matrix.
     */
    this.setTransform = function (layer, a, b, c, d, e, f) {
        scheduleTask(function __display_setTransform() {
            layer.setTransform(a, b, c, d, e, f);
        });
    };

    /**
     * Applies the given affine transform (defined with six values from the
     * transform's matrix).
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to modify.
     *
     * @param {!number} a
     *     The first value in the affine transform's matrix.
     *
     * @param {!number} b
     *     The second value in the affine transform's matrix.
     *
     * @param {!number} c
     *     The third value in the affine transform's matrix.
     *
     * @param {!number} d
     *     The fourth value in the affine transform's matrix.
     *
     * @param {!number} e
     *     The fifth value in the affine transform's matrix.
     *
     * @param {!number} f
     *     The sixth value in the affine transform's matrix.
     *
     */
    this.transform = function (layer, a, b, c, d, e, f) {
        scheduleTask(function __display_transform() {
            layer.transform(a, b, c, d, e, f);
        });
    };

    /**
     * Sets the channel mask for future operations on this Layer.
     *
     * The channel mask is a Guacamole-specific compositing operation identifier
     * with a single bit representing each of four channels (in order): source
     * image where destination transparent, source where destination opaque,
     * destination where source transparent, and destination where source
     * opaque.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to modify.
     *
     * @param {!number} mask
     *     The channel mask for future operations on this Layer.
     */
    this.setChannelMask = function (layer, mask) {
        scheduleTask(function __display_setChannelMask() {
            layer.setChannelMask(mask);
        });
    };

    /**
     * Sets the miter limit for stroke operations using the miter join. This
     * limit is the maximum ratio of the size of the miter join to the stroke
     * width. If this ratio is exceeded, the miter will not be drawn for that
     * joint of the path.
     *
     * @param {!Guacamole.Layer} layer
     *     The layer to modify.
     *
     * @param {!number} limit
     *     The miter limit for stroke operations using the miter join.
     */
    this.setMiterLimit = function (layer, limit) {
        scheduleTask(function __display_setMiterLimit() {
            layer.setMiterLimit(limit);
        });
    };

    /**
     * Removes the given layer container entirely, such that it is no longer
     * contained within its parent layer, if any.
     *
     * @param {!Guacamole.Display.VisibleLayer} layer
     *     The layer being removed from its parent.
     */
    this.dispose = function dispose(layer) {
        scheduleTask(function disposeLayer() {
            layer.dispose();
        });
    };

    /**
     * Applies the given affine transform (defined with six values from the
     * transform's matrix) to the given layer.
     *
     * @param {!Guacamole.Display.VisibleLayer} layer
     *     The layer being distorted.
     *
     * @param {!number} a
     *     The first value in the affine transform's matrix.
     *
     * @param {!number} b
     *     The second value in the affine transform's matrix.
     *
     * @param {!number} c
     *     The third value in the affine transform's matrix.
     *
     * @param {!number} d
     *     The fourth value in the affine transform's matrix.
     *
     * @param {!number} e
     *     The fifth value in the affine transform's matrix.
     *
     * @param {!number} f
     *     The sixth value in the affine transform's matrix.
     */
    this.distort = function distort(layer, a, b, c, d, e, f) {
        scheduleTask(function distortLayer() {
            layer.distort(a, b, c, d, e, f);
        });
    };

    /**
     * Moves the upper-left corner of the given layer to the given X and Y
     * coordinate, sets the Z stacking order, and reparents the layer
     * to the given parent layer.
     *
     * @param {!Guacamole.Display.VisibleLayer} layer
     *     The layer being moved.
     *
     * @param {!Guacamole.Display.VisibleLayer} parent
     *     The parent to set.
     *
     * @param {!number} x
     *     The X coordinate to move to.
     *
     * @param {!number} y
     *     The Y coordinate to move to.
     *
     * @param {!number} z
     *     The Z coordinate to move to.
     */
    this.move = function move(layer, parent, x, y, z) {
        scheduleTask(function moveLayer() {
            layer.move(parent, x, y, z);
        });
    };

    /**
     * Sets the opacity of the given layer to the given value, where 255 is
     * fully opaque and 0 is fully transparent.
     *
     * @param {!Guacamole.Display.VisibleLayer} layer
     *     The layer whose opacity should be set.
     *
     * @param {!number} alpha
     *     The opacity to set.
     */
    this.shade = function shade(layer, alpha) {
        scheduleTask(function shadeLayer() {
            layer.shade(alpha);
        });
    };

    /**
     * Sets the scale of the client display element such that it renders at
     * a relatively smaller or larger size, without affecting the true
     * resolution of the display.
     *
     * @param {!number} scale
     *     The scale to resize to, where 1.0 is normal size (1:1 scale).
     */
    this.scale = function (scale) {
        display.style.transform =
            display.style.WebkitTransform =
            display.style.MozTransform =
            display.style.OTransform =
            display.style.msTransform =
                'scale(' + scale + ',' + scale + ')';

        displayScale = scale;

        // Update bounds size
        bounds.style.width = displayWidth * displayScale + 'px';
        bounds.style.height = displayHeight * displayScale + 'px';
    };

    /**
     * Returns the scale of the display.
     *
     * @return {!number}
     *     The scale of the display.
     */
    this.getScale = function () {
        return displayScale;
    };

    /**
     * Returns a canvas element containing the entire display, with all child
     * layers composited within.
     *
     * @return {!HTMLCanvasElement}
     *     A new canvas element containing a copy of the display.
     */
    this.flatten = function () {
        // Get destination canvas
        var canvas = document.createElement('canvas');
        canvas.width = default_layer.width;
        canvas.height = default_layer.height;

        var context = canvas.getContext('2d');

        // Returns sorted array of children
        function get_children(layer) {
            // Build array of children
            var children = [];
            for (var index in layer.children) children.push(layer.children[index]);

            // Sort
            children.sort(function children_comparator(a, b) {
                // Compare based on Z order
                var diff = a.z - b.z;
                if (diff !== 0) return diff;

                // If Z order identical, use document order
                var a_element = a.getElement();
                var b_element = b.getElement();
                var position = b_element.compareDocumentPosition(a_element);

                if (position & Node.DOCUMENT_POSITION_PRECEDING) return -1;
                if (position & Node.DOCUMENT_POSITION_FOLLOWING) return 1;

                // Otherwise, assume same
                return 0;
            });

            // Done
            return children;
        }

        // Draws the contents of the given layer at the given coordinates
        function draw_layer(layer, x, y) {
            // Draw layer
            if (layer.width > 0 && layer.height > 0) {
                // Save and update alpha
                var initial_alpha = context.globalAlpha;
                context.globalAlpha *= layer.alpha / 255.0;

                // Copy data
                context.drawImage(layer.getCanvas(), x, y);

                // Draw all children
                var children = get_children(layer);
                for (var i = 0; i < children.length; i++) {
                    var child = children[i];
                    draw_layer(child, x + child.x, y + child.y);
                }

                // Restore alpha
                context.globalAlpha = initial_alpha;
            }
        }

        // Draw default layer and all children
        draw_layer(default_layer, 0, 0);

        // Return new canvas copy
        return canvas;
    };
};

/**
 * Simple container for Guacamole.Layer, allowing layers to be easily
 * repositioned and nested. This allows certain operations to be accelerated
 * through DOM manipulation, rather than raster operations.
 *
 * @constructor
 * @augments Guacamole.Layer
 * @param {!number} width
 *     The width of the Layer, in pixels. The canvas element backing this Layer
 *     will be given this width.
 *
 * @param {!number} height
 *     The height of the Layer, in pixels. The canvas element backing this
 *     Layer will be given this height.
 */
Guacamole.Display.VisibleLayer = function (width, height) {
    Guacamole.Layer.apply(this, [width, height]);

    /**
     * Reference to this layer.
     *
     * @private
     * @type {!Guacamole.Display.Layer}
     */
    var layer = this;

    /**
     * Identifier which uniquely identifies this layer. This is COMPLETELY
     * UNRELATED to the index of the underlying layer, which is specific
     * to the Guacamole protocol, and not relevant at this level.
     *
     * @private
     * @type {!number}
     */
    this.__unique_id = Guacamole.Display.VisibleLayer.__next_id++;

    /**
     * The opacity of the layer container, where 255 is fully opaque and 0 is
     * fully transparent.
     *
     * @type {!number}
     */
    this.alpha = 0xff;

    /**
     * X coordinate of the upper-left corner of this layer container within
     * its parent, in pixels.
     *
     * @type {!number}
     */
    this.x = 0;

    /**
     * Y coordinate of the upper-left corner of this layer container within
     * its parent, in pixels.
     *
     * @type {!number}
     */
    this.y = 0;

    /**
     * Z stacking order of this layer relative to other sibling layers.
     *
     * @type {!number}
     */
    this.z = 0;

    /**
     * The affine transformation applied to this layer container. Each element
     * corresponds to a value from the transformation matrix, with the first
     * three values being the first row, and the last three values being the
     * second row. There are six values total.
     *
     * @type {!number[]}
     */
    this.matrix = [1, 0, 0, 1, 0, 0];

    /**
     * The parent layer container of this layer, if any.
     * @type {Guacamole.Display.VisibleLayer}
     */
    this.parent = null;

    /**
     * Set of all children of this layer, indexed by layer index. This object
     * will have one property per child.
     *
     * @type {!Object.<number, Guacamole.Display.VisibleLayer>}
     */
    this.children = {};

    // Set layer position
    var canvas = layer.getCanvas();
    canvas.style.position = 'absolute';
    canvas.style.left = '0px';
    canvas.style.top = '0px';

    // Create div with given size
    var div = document.createElement('div');
    div.appendChild(canvas);
    div.style.width = width + 'px';
    div.style.height = height + 'px';
    div.style.position = 'absolute';
    div.style.left = '0px';
    div.style.top = '0px';
    div.style.overflow = 'hidden';

    /**
     * Superclass resize() function.
     * @private
     */
    var __super_resize = this.resize;

    this.resize = function (width, height) {
        // Resize containing div
        div.style.width = width + 'px';
        div.style.height = height + 'px';

        __super_resize(width, height);
    };

    /**
     * Returns the element containing the canvas and any other elements
     * associated with this layer.
     *
     * @returns {!Element}
     *     The element containing this layer's canvas.
     */
    this.getElement = function () {
        return div;
    };

    /**
     * The translation component of this layer's transform.
     *
     * @private
     * @type {!string}
     */
    var translate = 'translate(0px, 0px)'; // (0, 0)

    /**
     * The arbitrary matrix component of this layer's transform.
     *
     * @private
     * @type {!string}
     */
    var matrix = 'matrix(1, 0, 0, 1, 0, 0)'; // Identity

    /**
     * Moves the upper-left corner of this layer to the given X and Y
     * coordinate.
     *
     * @param {!number} x
     *     The X coordinate to move to.
     *
     * @param {!number} y
     *     The Y coordinate to move to.
     */
    this.translate = function (x, y) {
        layer.x = x;
        layer.y = y;

        // Generate translation
        translate = 'translate(' + x + 'px,' + y + 'px)';

        // Set layer transform
        div.style.transform = div.style.WebkitTransform = div.style.MozTransform = div.style.OTransform = div.style.msTransform = translate + ' ' + matrix;
    };

    /**
     * Moves the upper-left corner of this VisibleLayer to the given X and Y
     * coordinate, sets the Z stacking order, and reparents this VisibleLayer
     * to the given VisibleLayer.
     *
     * @param {!Guacamole.Display.VisibleLayer} parent
     *     The parent to set.
     *
     * @param {!number} x
     *     The X coordinate to move to.
     *
     * @param {!number} y
     *     The Y coordinate to move to.
     *
     * @param {!number} z
     *     The Z coordinate to move to.
     */
    this.move = function (parent, x, y, z) {
        // Set parent if necessary
        if (layer.parent !== parent) {
            // Maintain relationship
            if (layer.parent) delete layer.parent.children[layer.__unique_id];
            layer.parent = parent;
            parent.children[layer.__unique_id] = layer;

            // Reparent element
            var parent_element = parent.getElement();
            parent_element.appendChild(div);
        }

        // Set location
        layer.translate(x, y);
        layer.z = z;
        div.style.zIndex = z;
    };

    /**
     * Sets the opacity of this layer to the given value, where 255 is fully
     * opaque and 0 is fully transparent.
     *
     * @param {!number} a
     *     The opacity to set.
     */
    this.shade = function (a) {
        layer.alpha = a;
        div.style.opacity = a / 255.0;
    };

    /**
     * Removes this layer container entirely, such that it is no longer
     * contained within its parent layer, if any.
     */
    this.dispose = function () {
        // Remove from parent container
        if (layer.parent) {
            delete layer.parent.children[layer.__unique_id];
            layer.parent = null;
        }

        // Remove from parent element
        if (div.parentNode) div.parentNode.removeChild(div);
    };

    /**
     * Applies the given affine transform (defined with six values from the
     * transform's matrix).
     *
     * @param {!number} a
     *     The first value in the affine transform's matrix.
     *
     * @param {!number} b
     *     The second value in the affine transform's matrix.
     *
     * @param {!number} c
     *     The third value in the affine transform's matrix.
     *
     * @param {!number} d
     *     The fourth value in the affine transform's matrix.
     *
     * @param {!number} e
     *     The fifth value in the affine transform's matrix.
     *
     * @param {!number} f
     *     The sixth value in the affine transform's matrix.
     */
    this.distort = function (a, b, c, d, e, f) {
        // Store matrix
        layer.matrix = [a, b, c, d, e, f];

        // Generate matrix transformation
        matrix =
            /* a c e
             * b d f
             * 0 0 1
             */

            'matrix(' + a + ',' + b + ',' + c + ',' + d + ',' + e + ',' + f + ')';

        // Set layer transform
        div.style.transform = div.style.WebkitTransform = div.style.MozTransform = div.style.OTransform = div.style.msTransform = translate + ' ' + matrix;
    };
};

/**
 * The next identifier to be assigned to the layer container. This identifier
 * uniquely identifies each VisibleLayer, but is unrelated to the index of
 * the layer, which exists at the protocol/client level only.
 *
 * @private
 * @type {!number}
 */
Guacamole.Display.VisibleLayer.__next_id = 0;

/**
 * A set of Guacamole display performance statistics, describing the speed at
 * which the remote desktop, Guacamole server, and Guacamole client are
 * rendering frames.
 *
 * @constructor
 * @param {Guacamole.Display.Statistics|Object} [template={}]
 *     The object whose properties should be copied within the new
 *     Guacamole.Display.Statistics.
 */
Guacamole.Display.Statistics = function Statistics(template) {
    template = template || {};

    /**
     * The amount of time that the Guacamole client is taking to render
     * individual frames, in milliseconds, if known. If this value is unknown,
     * such as if the there are insufficient frame statistics recorded to
     * calculate this value, this will be null.
     *
     * @type {?number}
     */
    this.processingLag = template.processingLag;

    /**
     * The framerate of the remote desktop currently being viewed within the
     * relevant Gucamole.Display, independent of Guacamole, in frames per
     * second. This represents the speed at which the remote desktop is
     * producing frame data for the Guacamole server to consume. If this
     * value is unknown, such as if the remote desktop server does not actually
     * define frame boundaries, this will be null.
     *
     * @type {?number}
     */
    this.desktopFps = template.desktopFps;

    /**
     * The rate at which the Guacamole server is generating frames for the
     * Guacamole client to consume, in frames per second. If the Guacamole
     * server is correctly adjusting for variance in client/browser processing
     * power, this rate should closely match the client rate, and should remain
     * independent of any network latency. If this value is unknown, such as if
     * the there are insufficient frame statistics recorded to calculate this
     * value, this will be null.
     *
     * @type {?number}
     */
    this.serverFps = template.serverFps;

    /**
     * The rate at which the Guacamole client is consuming frames generated by
     * the Guacamole server, in frames per second. If the Guacamole server is
     * correctly adjusting for variance in client/browser processing power,
     * this rate should closely match the server rate, regardless of any
     * latency on the network between the server and client. If this value is
     * unknown, such as if the there are insufficient frame statistics recorded
     * to calculate this value, this will be null.
     *
     * @type {?number}
     */
    this.clientFps = template.clientFps;

    /**
     * The rate at which the Guacamole server is dropping or combining frames
     * received from the remote desktop server to compensate for variance in
     * client/browser processing power, in frames per second. This value may
     * also be non-zero if the server is compensating for variances in its own
     * processing power, or relative slowness in image compression vs. the rate
     * that inbound frames are received. If this value is unknown, such as if
     * the remote desktop server does not actually define frame boundaries,
     * this will be null.
     */
    this.dropRate = template.dropRate;
};

/**
 * An arbitrary event, emitted by a {@link Guacamole.Event.Target}. This object
 * should normally serve as the base class for a different object that is more
 * specific to the event type.
 *
 * @constructor
 * @param {!string} type
 *     The unique name of this event type.
 */
Guacamole.Event = function Event(type) {
    /**
     * The unique name of this event type.
     *
     * @type {!string}
     */
    this.type = type;

    /**
     * An arbitrary timestamp in milliseconds, indicating this event's
     * position in time relative to other events.
     *
     * @type {!number}
     */
    this.timestamp = new Date().getTime();

    /**
     * Returns the number of milliseconds elapsed since this event was created.
     *
     * @return {!number}
     *     The number of milliseconds elapsed since this event was created.
     */
    this.getAge = function getAge() {
        return new Date().getTime() - this.timestamp;
    };

    /**
     * Requests that the legacy event handler associated with this event be
     * invoked on the given event target. This function will be invoked
     * automatically by implementations of {@link Guacamole.Event.Target}
     * whenever {@link Guacamole.Event.Target#emit emit()} is invoked.
     * <p>
     * Older versions of Guacamole relied on single event handlers with the
     * prefix "on", such as "onmousedown" or "onkeyup". If a Guacamole.Event
     * implementation is replacing the event previously represented by one of
     * these handlers, this function gives the implementation the opportunity
     * to provide backward compatibility with the old handler.
     * <p>
     * Unless overridden, this function does nothing.
     *
     * @param {!Guacamole.Event.Target} eventTarget
     *     The {@link Guacamole.Event.Target} that emitted this event.
     */
    this.invokeLegacyHandler = function invokeLegacyHandler(eventTarget) {
        // Do nothing
    };
};

/**
 * A {@link Guacamole.Event} that may relate to one or more DOM events.
 * Continued propagation and default behavior of the related DOM events may be
 * prevented with {@link Guacamole.Event.DOMEvent#stopPropagation stopPropagation()}
 * and {@link Guacamole.Event.DOMEvent#preventDefault preventDefault()}
 * respectively.
 *
 * @constructor
 * @augments Guacamole.Event
 *
 * @param {!string} type
 *     The unique name of this event type.
 *
 * @param {Event|Event[]} [events=[]]
 *     The DOM events that are related to this event, if any. Future calls to
 *     {@link Guacamole.Event.DOMEvent#preventDefault preventDefault()} and
 *     {@link Guacamole.Event.DOMEvent#stopPropagation stopPropagation()} will
 *     affect these events.
 */
Guacamole.Event.DOMEvent = function DOMEvent(type, events) {
    Guacamole.Event.call(this, type);

    // Default to empty array
    events = events || [];

    // Automatically wrap non-array single Event in an array
    if (!Array.isArray(events)) events = [events];

    /**
     * Requests that the default behavior of related DOM events be prevented.
     * Whether this request will be honored by the browser depends on the
     * nature of those events and the timing of the request.
     */
    this.preventDefault = function preventDefault() {
        events.forEach(function applyPreventDefault(event) {
            if (event.preventDefault) event.preventDefault();
            event.returnValue = false;
        });
    };

    /**
     * Stops further propagation of related events through the DOM. Only events
     * that are directly related to this event will be stopped.
     */
    this.stopPropagation = function stopPropagation() {
        events.forEach(function applyStopPropagation(event) {
            event.stopPropagation();
        });
    };
};

/**
 * Convenience function for cancelling all further processing of a given DOM
 * event. Invoking this function prevents the default behavior of the event and
 * stops any further propagation.
 *
 * @param {!Event} event
 *     The DOM event to cancel.
 */
Guacamole.Event.DOMEvent.cancelEvent = function cancelEvent(event) {
    event.stopPropagation();
    if (event.preventDefault) event.preventDefault();
    event.returnValue = false;
};

/**
 * An object which can dispatch {@link Guacamole.Event} objects. Listeners
 * registered with {@link Guacamole.Event.Target#on on()} will automatically
 * be invoked based on the type of {@link Guacamole.Event} passed to
 * {@link Guacamole.Event.Target#dispatch dispatch()}. It is normally
 * subclasses of Guacamole.Event.Target that will dispatch events, and usages
 * of those subclasses that will catch dispatched events with on().
 *
 * @constructor
 */
Guacamole.Event.Target = function Target() {
    /**
     * A callback function which handles an event dispatched by an event
     * target.
     *
     * @callback Guacamole.Event.Target~listener
     * @param {!Guacamole.Event} event
     *     The event that was dispatched.
     *
     * @param {!Guacamole.Event.Target} target
     *     The object that dispatched the event.
     */

    /**
     * All listeners (callback functions) registered for each event type passed
     * to {@link Guacamole.Event.Targer#on on()}.
     *
     * @private
     * @type {!Object.<string, Guacamole.Event.Target~listener[]>}
     */
    var listeners = {};

    /**
     * Registers a listener for events having the given type, as dictated by
     * the {@link Guacamole.Event#type type} property of {@link Guacamole.Event}
     * provided to {@link Guacamole.Event.Target#dispatch dispatch()}.
     *
     * @param {!string} type
     *     The unique name of this event type.
     *
     * @param {!Guacamole.Event.Target~listener} listener
     *     The function to invoke when an event having the given type is
     *     dispatched. The {@link Guacamole.Event} object provided to
     *     {@link Guacamole.Event.Target#dispatch dispatch()} will be passed to
     *     this function, along with the dispatching Guacamole.Event.Target.
     */
    this.on = function on(type, listener) {
        var relevantListeners = listeners[type];
        if (!relevantListeners) listeners[type] = relevantListeners = [];

        relevantListeners.push(listener);
    };

    /**
     * Registers a listener for events having the given types, as dictated by
     * the {@link Guacamole.Event#type type} property of {@link Guacamole.Event}
     * provided to {@link Guacamole.Event.Target#dispatch dispatch()}.
     * <p>
     * Invoking this function is equivalent to manually invoking
     * {@link Guacamole.Event.Target#on on()} for each of the provided types.
     *
     * @param {!string[]} types
     *     The unique names of the event types to associate with the given
     *     listener.
     *
     * @param {!Guacamole.Event.Target~listener} listener
     *     The function to invoke when an event having any of the given types
     *     is dispatched. The {@link Guacamole.Event} object provided to
     *     {@link Guacamole.Event.Target#dispatch dispatch()} will be passed to
     *     this function, along with the dispatching Guacamole.Event.Target.
     */
    this.onEach = function onEach(types, listener) {
        types.forEach(function addListener(type) {
            this.on(type, listener);
        }, this);
    };

    /**
     * Dispatches the given event, invoking all event handlers registered with
     * this Guacamole.Event.Target for that event's
     * {@link Guacamole.Event#type type}.
     *
     * @param {!Guacamole.Event} event
     *     The event to dispatch.
     */
    this.dispatch = function dispatch(event) {
        // Invoke any relevant legacy handler for the event
        event.invokeLegacyHandler(this);

        // Invoke all registered listeners
        var relevantListeners = listeners[event.type];
        if (relevantListeners) {
            for (var i = 0; i < relevantListeners.length; i++) {
                relevantListeners[i](event, this);
            }
        }
    };

    /**
     * Unregisters a listener that was previously registered with
     * {@link Guacamole.Event.Target#on on()} or
     * {@link Guacamole.Event.Target#onEach onEach()}. If no such listener was
     * registered, this function has no effect. If multiple copies of the same
     * listener were registered, the first listener still registered will be
     * removed.
     *
     * @param {!string} type
     *     The unique name of the event type handled by the listener being
     *     removed.
     *
     * @param {!Guacamole.Event.Target~listener} listener
     *     The listener function previously provided to
     *     {@link Guacamole.Event.Target#on on()}or
     *     {@link Guacamole.Event.Target#onEach onEach()}.
     *
     * @returns {!boolean}
     *     true if the specified listener was removed, false otherwise.
     */
    this.off = function off(type, listener) {
        var relevantListeners = listeners[type];
        if (!relevantListeners) return false;

        for (var i = 0; i < relevantListeners.length; i++) {
            if (relevantListeners[i] === listener) {
                relevantListeners.splice(i, 1);
                return true;
            }
        }

        return false;
    };

    /**
     * Unregisters listeners that were previously registered with
     * {@link Guacamole.Event.Target#on on()} or
     * {@link Guacamole.Event.Target#onEach onEach()}. If no such listeners
     * were registered, this function has no effect. If multiple copies of the
     * same listener were registered for the same event type, the first
     * listener still registered will be removed.
     * <p>
     * Invoking this function is equivalent to manually invoking
     * {@link Guacamole.Event.Target#off off()} for each of the provided types.
     *
     * @param {!string[]} types
     *     The unique names of the event types handled by the listeners being
     *     removed.
     *
     * @param {!Guacamole.Event.Target~listener} listener
     *     The listener function previously provided to
     *     {@link Guacamole.Event.Target#on on()} or
     *     {@link Guacamole.Event.Target#onEach onEach()}.
     *
     * @returns {!boolean}
     *     true if any of the specified listeners were removed, false
     *     otherwise.
     */
    this.offEach = function offEach(types, listener) {
        var changed = false;

        types.forEach(function removeListener(type) {
            changed |= this.off(type, listener);
        }, this);

        return changed;
    };
};

/**
 * A hidden input field which attempts to keep itself focused at all times,
 * except when another input field has been intentionally focused, whether
 * programatically or by the user. The actual underlying input field, returned
 * by getElement(), may be used as a reliable source of keyboard-related events,
 * particularly composition and input events which may require a focused input
 * field to be dispatched at all.
 *
 * @constructor
 */
Guacamole.InputSink = function InputSink() {
    /**
     * Reference to this instance of Guacamole.InputSink.
     *
     * @private
     * @type {!Guacamole.InputSink}
     */
    var sink = this;

    /**
     * The underlying input field, styled to be invisible.
     *
     * @private
     * @type {!Element}
     */
    var field = document.createElement('textarea');
    field.style.position = 'fixed';
    field.style.outline = 'none';
    field.style.border = 'none';
    field.style.margin = '0';
    field.style.padding = '0';
    field.style.height = '0';
    field.style.width = '0';
    field.style.left = '0';
    field.style.bottom = '0';
    field.style.resize = 'none';
    field.style.background = 'transparent';
    field.style.color = 'transparent';

    // Keep field clear when modified via normal keypresses
    field.addEventListener(
        'keypress',
        function clearKeypress(e) {
            field.value = '';
        },
        false
    );

    // Keep field clear when modofied via composition events
    field.addEventListener(
        'compositionend',
        function clearCompletedComposition(e) {
            if (e.data) field.value = '';
        },
        false
    );

    // Keep field clear when modofied via input events
    field.addEventListener(
        'input',
        function clearCompletedInput(e) {
            if (e.data && !e.isComposing) field.value = '';
        },
        false
    );

    // Whenever focus is gained, automatically click to ensure cursor is
    // actually placed within the field (the field may simply be highlighted or
    // outlined otherwise)
    field.addEventListener(
        'focus',
        function focusReceived() {
            window.setTimeout(function deferRefocus() {
                field.click();
                field.select();
            }, 0);
        },
        true
    );

    /**
     * Attempts to focus the underlying input field. The focus attempt occurs
     * asynchronously, and may silently fail depending on browser restrictions.
     */
    this.focus = function focus() {
        window.setTimeout(function deferRefocus() {
            field.focus(); // Focus must be deferred to work reliably across browsers
        }, 0);
    };

    /**
     * Returns the underlying input field. This input field MUST be manually
     * added to the DOM for the Guacamole.InputSink to have any effect.
     *
     * @returns {!Element}
     *     The underlying input field.
     */
    this.getElement = function getElement() {
        return field;
    };

    // Automatically refocus input sink if part of DOM
    document.addEventListener(
        'keydown',
        function refocusSink(e) {
            // Do not refocus if focus is on an input field
            var focused = document.activeElement;
            if (focused && focused !== document.body) {
                // Only consider focused input fields which are actually visible
                var rect = focused.getBoundingClientRect();
                if (rect.left + rect.width > 0 && rect.top + rect.height > 0) return;
            }

            // Refocus input sink instead of handling click
            sink.focus();
        },
        true
    );
};

/**
 * An input stream abstraction used by the Guacamole client to facilitate
 * transfer of files or other binary data.
 *
 * @constructor
 * @param {!Guacamole.Client} client
 *     The client owning this stream.
 *
 * @param {!number} index
 *     The index of this stream.
 */
Guacamole.InputStream = function (client, index) {
    /**
     * Reference to this stream.
     *
     * @private
     * @type {!Guacamole.InputStream}
     */
    var guac_stream = this;

    /**
     * The index of this stream.
     *
     * @type {!number}
     */
    this.index = index;

    /**
     * Called when a blob of data is received.
     *
     * @event
     * @param {!string} data
     *     The received base64 data.
     */
    this.onblob = null;

    /**
     * Called when this stream is closed.
     *
     * @event
     */
    this.onend = null;

    /**
     * Acknowledges the receipt of a blob.
     *
     * @param {!string} message
     *     A human-readable message describing the error or status.
     *
     * @param {!number} code
     *     The error code, if any, or 0 for success.
     */
    this.sendAck = function (message, code) {
        client.sendAck(guac_stream.index, message, code);
    };
};

/**
 * Integer pool which returns consistently increasing integers while integers
 * are in use, and previously-used integers when possible.
 * @constructor
 */
Guacamole.IntegerPool = function () {
    /**
     * Reference to this integer pool.
     *
     * @private
     */
    var guac_pool = this;

    /**
     * Array of available integers.
     *
     * @private
     * @type {!number[]}
     */
    var pool = [];

    /**
     * The next integer to return if no more integers remain.
     *
     * @type {!number}
     */
    this.next_int = 0;

    /**
     * Returns the next available integer in the pool. If possible, a previously
     * used integer will be returned.
     *
     * @return {!number}
     *     The next available integer.
     */
    this.next = function () {
        // If free'd integers exist, return one of those
        if (pool.length > 0) return pool.shift();

        // Otherwise, return a new integer
        return guac_pool.next_int++;
    };

    /**
     * Frees the given integer, allowing it to be reused.
     *
     * @param {!number} integer
     *     The integer to free.
     */
    this.free = function (integer) {
        pool.push(integer);
    };
};

/**
 * A reader which automatically handles the given input stream, assembling all
 * received blobs into a JavaScript object by appending them to each other, in
 * order, and decoding the result as JSON. Note that this object will overwrite
 * any installed event handlers on the given Guacamole.InputStream.
 *
 * @constructor
 * @param {Guacamole.InputStream} stream
 *     The stream that JSON will be read from.
 */
Guacamole.JSONReader = function guacamoleJSONReader(stream) {
    /**
     * Reference to this Guacamole.JSONReader.
     *
     * @private
     * @type {!Guacamole.JSONReader}
     */
    var guacReader = this;

    /**
     * Wrapped Guacamole.StringReader.
     *
     * @private
     * @type {!Guacamole.StringReader}
     */
    var stringReader = new Guacamole.StringReader(stream);

    /**
     * All JSON read thus far.
     *
     * @private
     * @type {!string}
     */
    var json = '';

    /**
     * Returns the current length of this Guacamole.JSONReader, in characters.
     *
     * @return {!number}
     *     The current length of this Guacamole.JSONReader.
     */
    this.getLength = function getLength() {
        return json.length;
    };

    /**
     * Returns the contents of this Guacamole.JSONReader as a JavaScript
     * object.
     *
     * @return {object}
     *     The contents of this Guacamole.JSONReader, as parsed from the JSON
     *     contents of the input stream.
     */
    this.getJSON = function getJSON() {
        return JSON.parse(json);
    };

    // Append all received text
    stringReader.ontext = function ontext(text) {
        // Append received text
        json += text;

        // Call handler, if present
        if (guacReader.onprogress) guacReader.onprogress(text.length);
    };

    // Simply call onend when end received
    stringReader.onend = function onend() {
        if (guacReader.onend) guacReader.onend();
    };

    /**
     * Fired once for every blob of data received.
     *
     * @event
     * @param {!number} length
     *     The number of characters received.
     */
    this.onprogress = null;

    /**
     * Fired once this stream is finished and no further data will be written.
     *
     * @event
     */
    this.onend = null;
};

/**
 * Provides cross-browser and cross-keyboard keyboard for a specific element.
 * Browser and keyboard layout variation is abstracted away, providing events
 * which represent keys as their corresponding X11 keysym.
 *
 * @constructor
 * @param {Element|Document} [element]
 *    The Element to use to provide keyboard events. If omitted, at least one
 *    Element must be manually provided through the listenTo() function for
 *    the Guacamole.Keyboard instance to have any effect.
 */
Guacamole.Keyboard = function Keyboard(element) {
    /**
     * Reference to this Guacamole.Keyboard.
     *
     * @private
     * @type {!Guacamole.Keyboard}
     */
    var guac_keyboard = this;

    /**
     * An integer value which uniquely identifies this Guacamole.Keyboard
     * instance with respect to other Guacamole.Keyboard instances.
     *
     * @private
     * @type {!number}
     */
    var guacKeyboardID = Guacamole.Keyboard._nextID++;

    /**
     * The name of the property which is added to event objects via markEvent()
     * to note that they have already been handled by this Guacamole.Keyboard.
     *
     * @private
     * @constant
     * @type {!string}
     */
    var EVENT_MARKER = '_GUAC_KEYBOARD_HANDLED_BY_' + guacKeyboardID;

    /**
     * Fired whenever the user presses a key with the element associated
     * with this Guacamole.Keyboard in focus.
     *
     * @event
     * @param {!number} keysym
     *     The keysym of the key being pressed.
     *
     * @return {!boolean}
     *     true if the key event should be allowed through to the browser,
     *     false otherwise.
     */
    this.onkeydown = null;

    /**
     * Fired whenever the user releases a key with the element associated
     * with this Guacamole.Keyboard in focus.
     *
     * @event
     * @param {!number} keysym
     *     The keysym of the key being released.
     */
    this.onkeyup = null;

    /**
     * Set of known platform-specific or browser-specific quirks which must be
     * accounted for to properly interpret key events, even if the only way to
     * reliably detect that quirk is to platform/browser-sniff.
     *
     * @private
     * @type {!Object.<string, boolean>}
     */
    var quirks = {
        /**
         * Whether keyup events are universally unreliable.
         *
         * @type {!boolean}
         */
        keyupUnreliable: false,

        /**
         * Whether the Alt key is actually a modifier for typable keys and is
         * thus never used for keyboard shortcuts.
         *
         * @type {!boolean}
         */
        altIsTypableOnly: false,

        /**
         * Whether we can rely on receiving a keyup event for the Caps Lock
         * key.
         *
         * @type {!boolean}
         */
        capsLockKeyupUnreliable: false,
    };

    // Set quirk flags depending on platform/browser, if such information is
    // available
    if (navigator && navigator.platform) {
        // All keyup events are unreliable on iOS (sadly)
        if (navigator.platform.match(/ipad|iphone|ipod/i)) quirks.keyupUnreliable = true;
        // The Alt key on Mac is never used for keyboard shortcuts, and the
        // Caps Lock key never dispatches keyup events
        else if (navigator.platform.match(/^mac/i)) {
            quirks.altIsTypableOnly = true;
            quirks.capsLockKeyupUnreliable = true;
        }
    }

    /**
     * A key event having a corresponding timestamp. This event is non-specific.
     * Its subclasses should be used instead when recording specific key
     * events.
     *
     * @private
     * @constructor
     * @param {KeyboardEvent} [orig]
     *     The relevant DOM keyboard event.
     */
    var KeyEvent = function KeyEvent(orig) {
        /**
         * Reference to this key event.
         *
         * @private
         * @type {!KeyEvent}
         */
        var key_event = this;

        /**
         * The JavaScript key code of the key pressed. For most events (keydown
         * and keyup), this is a scancode-like value related to the position of
         * the key on the US English "Qwerty" keyboard. For keypress events,
         * this is the Unicode codepoint of the character that would be typed
         * by the key pressed.
         *
         * @type {!number}
         */
        this.keyCode = orig ? orig.which || orig.keyCode : 0;

        /**
         * The legacy DOM3 "keyIdentifier" of the key pressed, as defined at:
         * http://www.w3.org/TR/2009/WD-DOM-Level-3-Events-20090908/#events-Events-KeyboardEvent
         *
         * @type {!string}
         */
        this.keyIdentifier = orig && orig.keyIdentifier;

        /**
         * The standard name of the key pressed, as defined at:
         * http://www.w3.org/TR/DOM-Level-3-Events/#events-KeyboardEvent
         *
         * @type {!string}
         */
        this.key = orig && orig.key;

        /**
         * The location on the keyboard corresponding to the key pressed, as
         * defined at:
         * http://www.w3.org/TR/DOM-Level-3-Events/#events-KeyboardEvent
         *
         * @type {!number}
         */
        this.location = orig ? getEventLocation(orig) : 0;

        /**
         * The state of all local keyboard modifiers at the time this event was
         * received.
         *
         * @type {!Guacamole.Keyboard.ModifierState}
         */
        this.modifiers = orig ? Guacamole.Keyboard.ModifierState.fromKeyboardEvent(orig) : new Guacamole.Keyboard.ModifierState();

        /**
         * An arbitrary timestamp in milliseconds, indicating this event's
         * position in time relative to other events.
         *
         * @type {!number}
         */
        this.timestamp = new Date().getTime();

        /**
         * Whether the default action of this key event should be prevented.
         *
         * @type {!boolean}
         */
        this.defaultPrevented = false;

        /**
         * The keysym of the key associated with this key event, as determined
         * by a best-effort guess using available event properties and keyboard
         * state.
         *
         * @type {number}
         */
        this.keysym = null;

        /**
         * Whether the keysym value of this key event is known to be reliable.
         * If false, the keysym may still be valid, but it's only a best guess,
         * and future key events may be a better source of information.
         *
         * @type {!boolean}
         */
        this.reliable = false;

        /**
         * Returns the number of milliseconds elapsed since this event was
         * received.
         *
         * @return {!number}
         *     The number of milliseconds elapsed since this event was
         *     received.
         */
        this.getAge = function () {
            return new Date().getTime() - key_event.timestamp;
        };
    };

    /**
     * Information related to the pressing of a key, which need not be a key
     * associated with a printable character. The presence or absence of any
     * information within this object is browser-dependent.
     *
     * @private
     * @constructor
     * @augments Guacamole.Keyboard.KeyEvent
     * @param {!KeyboardEvent} orig
     *     The relevant DOM "keydown" event.
     */
    var KeydownEvent = function KeydownEvent(orig) {
        // We extend KeyEvent
        KeyEvent.call(this, orig);

        // If key is known from keyCode or DOM3 alone, use that
        this.keysym = keysym_from_key_identifier(this.key, this.location) || keysym_from_keycode(this.keyCode, this.location);

        /**
         * Whether the keyup following this keydown event is known to be
         * reliable. If false, we cannot rely on the keyup event to occur.
         *
         * @type {!boolean}
         */
        this.keyupReliable = !quirks.keyupUnreliable;

        // DOM3 and keyCode are reliable sources if the corresponding key is
        // not a printable key
        if (this.keysym && !isPrintable(this.keysym)) this.reliable = true;

        // Use legacy keyIdentifier as a last resort, if it looks sane
        if (!this.keysym && key_identifier_sane(this.keyCode, this.keyIdentifier))
            this.keysym = keysym_from_key_identifier(this.keyIdentifier, this.location, this.modifiers.shift);

        // If a key is pressed while meta is held down, the keyup will
        // never be sent in Chrome (bug #108404)
        if (this.modifiers.meta && this.keysym !== 0xffe7 && this.keysym !== 0xffe8) this.keyupReliable = false;
        // We cannot rely on receiving keyup for Caps Lock on certain platforms
        else if (this.keysym === 0xffe5 && quirks.capsLockKeyupUnreliable) this.keyupReliable = false;

        // Determine whether default action for Alt+combinations must be prevented
        var prevent_alt = !this.modifiers.ctrl && !quirks.altIsTypableOnly;

        // Determine whether default action for Ctrl+combinations must be prevented
        var prevent_ctrl = !this.modifiers.alt;

        // We must rely on the (potentially buggy) keyIdentifier if preventing
        // the default action is important
        if ((prevent_ctrl && this.modifiers.ctrl) || (prevent_alt && this.modifiers.alt) || this.modifiers.meta || this.modifiers.hyper) this.reliable = true;

        // Record most recently known keysym by associated key code
        recentKeysym[this.keyCode] = this.keysym;
    };

    KeydownEvent.prototype = new KeyEvent();

    /**
     * Information related to the pressing of a key, which MUST be
     * associated with a printable character. The presence or absence of any
     * information within this object is browser-dependent.
     *
     * @private
     * @constructor
     * @augments Guacamole.Keyboard.KeyEvent
     * @param {!KeyboardEvent} orig
     *     The relevant DOM "keypress" event.
     */
    var KeypressEvent = function KeypressEvent(orig) {
        // We extend KeyEvent
        KeyEvent.call(this, orig);

        // Pull keysym from char code
        this.keysym = keysym_from_charcode(this.keyCode);

        // Keypress is always reliable
        this.reliable = true;
    };

    KeypressEvent.prototype = new KeyEvent();

    /**
     * Information related to the releasing of a key, which need not be a key
     * associated with a printable character. The presence or absence of any
     * information within this object is browser-dependent.
     *
     * @private
     * @constructor
     * @augments Guacamole.Keyboard.KeyEvent
     * @param {!KeyboardEvent} orig
     *     The relevant DOM "keyup" event.
     */
    var KeyupEvent = function KeyupEvent(orig) {
        // We extend KeyEvent
        KeyEvent.call(this, orig);

        // If key is known from keyCode or DOM3 alone, use that (keyCode is
        // still more reliable for keyup when dead keys are in use)
        this.keysym = keysym_from_keycode(this.keyCode, this.location) || keysym_from_key_identifier(this.key, this.location);

        // Fall back to the most recently pressed keysym associated with the
        // keyCode if the inferred key doesn't seem to actually be pressed
        if (!guac_keyboard.pressed[this.keysym]) this.keysym = recentKeysym[this.keyCode] || this.keysym;

        // Keyup is as reliable as it will ever be
        this.reliable = true;
    };

    KeyupEvent.prototype = new KeyEvent();

    /**
     * An array of recorded events, which can be instances of the private
     * KeydownEvent, KeypressEvent, and KeyupEvent classes.
     *
     * @private
     * @type {!KeyEvent[]}
     */
    var eventLog = [];

    /**
     * Map of known JavaScript keycodes which do not map to typable characters
     * to their X11 keysym equivalents.
     *
     * @private
     * @type {!Object.<number, number[]>}
     */
    var keycodeKeysyms = {
        8: [0xff08], // backspace
        9: [0xff09], // tab
        12: [0xff0b, 0xff0b, 0xff0b, 0xffb5], // clear       / KP 5
        13: [0xff0d], // enter
        16: [0xffe1, 0xffe1, 0xffe2], // shift
        17: [0xffe3, 0xffe3, 0xffe4], // ctrl
        18: [0xffe9, 0xffe9, 0xfe03], // alt
        19: [0xff13], // pause/break
        20: [0xffe5], // caps lock
        27: [0xff1b], // escape
        32: [0x0020], // space
        33: [0xff55, 0xff55, 0xff55, 0xffb9], // page up     / KP 9
        34: [0xff56, 0xff56, 0xff56, 0xffb3], // page down   / KP 3
        35: [0xff57, 0xff57, 0xff57, 0xffb1], // end         / KP 1
        36: [0xff50, 0xff50, 0xff50, 0xffb7], // home        / KP 7
        37: [0xff51, 0xff51, 0xff51, 0xffb4], // left arrow  / KP 4
        38: [0xff52, 0xff52, 0xff52, 0xffb8], // up arrow    / KP 8
        39: [0xff53, 0xff53, 0xff53, 0xffb6], // right arrow / KP 6
        40: [0xff54, 0xff54, 0xff54, 0xffb2], // down arrow  / KP 2
        45: [0xff63, 0xff63, 0xff63, 0xffb0], // insert      / KP 0
        46: [0xffff, 0xffff, 0xffff, 0xffae], // delete      / KP decimal
        91: [0xffe7], // left windows/command key (meta_l)
        92: [0xffe8], // right window/command key (meta_r)
        93: [0xff67], // menu key
        96: [0xffb0], // KP 0
        97: [0xffb1], // KP 1
        98: [0xffb2], // KP 2
        99: [0xffb3], // KP 3
        100: [0xffb4], // KP 4
        101: [0xffb5], // KP 5
        102: [0xffb6], // KP 6
        103: [0xffb7], // KP 7
        104: [0xffb8], // KP 8
        105: [0xffb9], // KP 9
        106: [0xffaa], // KP multiply
        107: [0xffab], // KP add
        109: [0xffad], // KP subtract
        110: [0xffae], // KP decimal
        111: [0xffaf], // KP divide
        112: [0xffbe], // f1
        113: [0xffbf], // f2
        114: [0xffc0], // f3
        115: [0xffc1], // f4
        116: [0xffc2], // f5
        117: [0xffc3], // f6
        118: [0xffc4], // f7
        119: [0xffc5], // f8
        120: [0xffc6], // f9
        121: [0xffc7], // f10
        122: [0xffc8], // f11
        123: [0xffc9], // f12
        144: [0xff7f], // num lock
        145: [0xff14], // scroll lock
        225: [0xfe03], // altgraph (iso_level3_shift)
    };

    /**
     * Map of known JavaScript keyidentifiers which do not map to typable
     * characters to their unshifted X11 keysym equivalents.
     *
     * @private
     * @type {!Object.<string, number[]>}
     */
    var keyidentifier_keysym = {
        Again: [0xff66],
        AllCandidates: [0xff3d],
        Alphanumeric: [0xff30],
        Alt: [0xffe9, 0xffe9, 0xfe03],
        Attn: [0xfd0e],
        AltGraph: [0xfe03],
        ArrowDown: [0xff54],
        ArrowLeft: [0xff51],
        ArrowRight: [0xff53],
        ArrowUp: [0xff52],
        Backspace: [0xff08],
        CapsLock: [0xffe5],
        Cancel: [0xff69],
        Clear: [0xff0b],
        Convert: [0xff21],
        Copy: [0xfd15],
        Crsel: [0xfd1c],
        CrSel: [0xfd1c],
        CodeInput: [0xff37],
        Compose: [0xff20],
        Control: [0xffe3, 0xffe3, 0xffe4],
        ContextMenu: [0xff67],
        Delete: [0xffff],
        Down: [0xff54],
        End: [0xff57],
        Enter: [0xff0d],
        EraseEof: [0xfd06],
        Escape: [0xff1b],
        Execute: [0xff62],
        Exsel: [0xfd1d],
        ExSel: [0xfd1d],
        F1: [0xffbe],
        F2: [0xffbf],
        F3: [0xffc0],
        F4: [0xffc1],
        F5: [0xffc2],
        F6: [0xffc3],
        F7: [0xffc4],
        F8: [0xffc5],
        F9: [0xffc6],
        F10: [0xffc7],
        F11: [0xffc8],
        F12: [0xffc9],
        F13: [0xffca],
        F14: [0xffcb],
        F15: [0xffcc],
        F16: [0xffcd],
        F17: [0xffce],
        F18: [0xffcf],
        F19: [0xffd0],
        F20: [0xffd1],
        F21: [0xffd2],
        F22: [0xffd3],
        F23: [0xffd4],
        F24: [0xffd5],
        Find: [0xff68],
        GroupFirst: [0xfe0c],
        GroupLast: [0xfe0e],
        GroupNext: [0xfe08],
        GroupPrevious: [0xfe0a],
        FullWidth: null,
        HalfWidth: null,
        HangulMode: [0xff31],
        Hankaku: [0xff29],
        HanjaMode: [0xff34],
        Help: [0xff6a],
        Hiragana: [0xff25],
        HiraganaKatakana: [0xff27],
        Home: [0xff50],
        Hyper: [0xffed, 0xffed, 0xffee],
        Insert: [0xff63],
        JapaneseHiragana: [0xff25],
        JapaneseKatakana: [0xff26],
        JapaneseRomaji: [0xff24],
        JunjaMode: [0xff38],
        KanaMode: [0xff2d],
        KanjiMode: [0xff21],
        Katakana: [0xff26],
        Left: [0xff51],
        Meta: [0xffe7, 0xffe7, 0xffe8],
        ModeChange: [0xff7e],
        NumLock: [0xff7f],
        PageDown: [0xff56],
        PageUp: [0xff55],
        Pause: [0xff13],
        Play: [0xfd16],
        PreviousCandidate: [0xff3e],
        PrintScreen: [0xff61],
        Redo: [0xff66],
        Right: [0xff53],
        RomanCharacters: null,
        Scroll: [0xff14],
        Select: [0xff60],
        Separator: [0xffac],
        Shift: [0xffe1, 0xffe1, 0xffe2],
        SingleCandidate: [0xff3c],
        Super: [0xffeb, 0xffeb, 0xffec],
        Tab: [0xff09],
        UIKeyInputDownArrow: [0xff54],
        UIKeyInputEscape: [0xff1b],
        UIKeyInputLeftArrow: [0xff51],
        UIKeyInputRightArrow: [0xff53],
        UIKeyInputUpArrow: [0xff52],
        Up: [0xff52],
        Undo: [0xff65],
        Win: [0xffe7, 0xffe7, 0xffe8],
        Zenkaku: [0xff28],
        ZenkakuHankaku: [0xff2a],
    };

    /**
     * All keysyms which should not repeat when held down.
     *
     * @private
     * @type {!Object.<number, boolean>}
     */
    var no_repeat = {
        0xfe03: true, // ISO Level 3 Shift (AltGr)
        0xffe1: true, // Left shift
        0xffe2: true, // Right shift
        0xffe3: true, // Left ctrl
        0xffe4: true, // Right ctrl
        0xffe5: true, // Caps Lock
        0xffe7: true, // Left meta
        0xffe8: true, // Right meta
        0xffe9: true, // Left alt
        0xffea: true, // Right alt
        0xffeb: true, // Left super/hyper
        0xffec: true, // Right super/hyper
    };

    /**
     * All modifiers and their states.
     *
     * @type {!Guacamole.Keyboard.ModifierState}
     */
    this.modifiers = new Guacamole.Keyboard.ModifierState();

    /**
     * The state of every key, indexed by keysym. If a particular key is
     * pressed, the value of pressed for that keysym will be true. If a key
     * is not currently pressed, it will not be defined.
     *
     * @type {!Object.<number, boolean>}
     */
    this.pressed = {};

    /**
     * The state of every key, indexed by keysym, for strictly those keys whose
     * status has been indirectly determined thorugh observation of other key
     * events. If a particular key is implicitly pressed, the value of
     * implicitlyPressed for that keysym will be true. If a key
     * is not currently implicitly pressed (the key is not pressed OR the state
     * of the key is explicitly known), it will not be defined.
     *
     * @private
     * @type {!Object.<number, boolean>}
     */
    var implicitlyPressed = {};

    /**
     * The last result of calling the onkeydown handler for each key, indexed
     * by keysym. This is used to prevent/allow default actions for key events,
     * even when the onkeydown handler cannot be called again because the key
     * is (theoretically) still pressed.
     *
     * @private
     * @type {!Object.<number, boolean>}
     */
    var last_keydown_result = {};

    /**
     * The keysym most recently associated with a given keycode when keydown
     * fired. This object maps keycodes to keysyms.
     *
     * @private
     * @type {!Object.<number, number>}
     */
    var recentKeysym = {};

    /**
     * Timeout before key repeat starts.
     *
     * @private
     * @type {number}
     */
    var key_repeat_timeout = null;

    /**
     * Interval which presses and releases the last key pressed while that
     * key is still being held down.
     *
     * @private
     * @type {number}
     */
    var key_repeat_interval = null;

    /**
     * Given an array of keysyms indexed by location, returns the keysym
     * for the given location, or the keysym for the standard location if
     * undefined.
     *
     * @private
     * @param {number[]} keysyms
     *     An array of keysyms, where the index of the keysym in the array is
     *     the location value.
     *
     * @param {!number} location
     *     The location on the keyboard corresponding to the key pressed, as
     *     defined at: http://www.w3.org/TR/DOM-Level-3-Events/#events-KeyboardEvent
     */
    var get_keysym = function get_keysym(keysyms, location) {
        if (!keysyms) return null;

        return keysyms[location] || keysyms[0];
    };

    /**
     * Returns true if the given keysym corresponds to a printable character,
     * false otherwise.
     *
     * @param {!number} keysym
     *     The keysym to check.
     *
     * @returns {!boolean}
     *     true if the given keysym corresponds to a printable character,
     *     false otherwise.
     */
    var isPrintable = function isPrintable(keysym) {
        // Keysyms with Unicode equivalents are printable
        return (keysym >= 0x00 && keysym <= 0xff) || (keysym & 0xffff0000) === 0x01000000;
    };

    function keysym_from_key_identifier(identifier, location, shifted) {
        if (!identifier) return null;

        var typedCharacter;

        // If identifier is U+xxxx, decode Unicode character
        var unicodePrefixLocation = identifier.indexOf('U+');
        if (unicodePrefixLocation >= 0) {
            var hex = identifier.substring(unicodePrefixLocation + 2);
            typedCharacter = String.fromCharCode(parseInt(hex, 16));
        }

        // If single character and not keypad, use that as typed character
        else if (identifier.length === 1 && location !== 3) typedCharacter = identifier;
        // Otherwise, look up corresponding keysym
        else return get_keysym(keyidentifier_keysym[identifier], location);

        // Alter case if necessary
        if (shifted === true) typedCharacter = typedCharacter.toUpperCase();
        else if (shifted === false) typedCharacter = typedCharacter.toLowerCase();

        // Get codepoint
        var codepoint = typedCharacter.charCodeAt(0);
        return keysym_from_charcode(codepoint);
    }

    function isControlCharacter(codepoint) {
        return codepoint <= 0x1f || (codepoint >= 0x7f && codepoint <= 0x9f);
    }

    function keysym_from_charcode(codepoint) {
        // Keysyms for control characters
        if (isControlCharacter(codepoint)) return 0xff00 | codepoint;

        // Keysyms for ASCII chars
        if (codepoint >= 0x0000 && codepoint <= 0x00ff) return codepoint;

        // Keysyms for Unicode
        if (codepoint >= 0x0100 && codepoint <= 0x10ffff) return 0x01000000 | codepoint;

        return null;
    }

    function keysym_from_keycode(keyCode, location) {
        return get_keysym(keycodeKeysyms[keyCode], location);
    }

    /**
     * Heuristically detects if the legacy keyIdentifier property of
     * a keydown/keyup event looks incorrectly derived. Chrome, and
     * presumably others, will produce the keyIdentifier by assuming
     * the keyCode is the Unicode codepoint for that key. This is not
     * correct in all cases.
     *
     * @private
     * @param {!number} keyCode
     *     The keyCode from a browser keydown/keyup event.
     *
     * @param {string} keyIdentifier
     *     The legacy keyIdentifier from a browser keydown/keyup event.
     *
     * @returns {!boolean}
     *     true if the keyIdentifier looks sane, false if the keyIdentifier
     *     appears incorrectly derived or is missing entirely.
     */
    var key_identifier_sane = function key_identifier_sane(keyCode, keyIdentifier) {
        // Missing identifier is not sane
        if (!keyIdentifier) return false;

        // Assume non-Unicode keyIdentifier values are sane
        var unicodePrefixLocation = keyIdentifier.indexOf('U+');
        if (unicodePrefixLocation === -1) return true;

        // If the Unicode codepoint isn't identical to the keyCode,
        // then the identifier is likely correct
        var codepoint = parseInt(keyIdentifier.substring(unicodePrefixLocation + 2), 16);
        if (keyCode !== codepoint) return true;

        // The keyCodes for A-Z and 0-9 are actually identical to their
        // Unicode codepoints
        if ((keyCode >= 65 && keyCode <= 90) || (keyCode >= 48 && keyCode <= 57)) return true;

        // The keyIdentifier does NOT appear sane
        return false;
    };

    /**
     * Marks a key as pressed, firing the keydown event if registered. Key
     * repeat for the pressed key will start after a delay if that key is
     * not a modifier. The return value of this function depends on the
     * return value of the keydown event handler, if any.
     *
     * @param {number} keysym
     *     The keysym of the key to press.
     *
     * @return {boolean}
     *     true if event should NOT be canceled, false otherwise.
     */
    this.press = function (keysym) {
        // Don't bother with pressing the key if the key is unknown
        if (keysym === null) return;

        // Only press if released
        if (!guac_keyboard.pressed[keysym]) {
            // Mark key as pressed
            guac_keyboard.pressed[keysym] = true;

            // Send key event
            if (guac_keyboard.onkeydown) {
                var result = guac_keyboard.onkeydown(keysym);
                last_keydown_result[keysym] = result;

                // Stop any current repeat
                window.clearTimeout(key_repeat_timeout);
                window.clearInterval(key_repeat_interval);

                // Repeat after a delay as long as pressed
                if (!no_repeat[keysym])
                    key_repeat_timeout = window.setTimeout(function () {
                        key_repeat_interval = window.setInterval(function () {
                            guac_keyboard.onkeyup(keysym);
                            guac_keyboard.onkeydown(keysym);
                        }, 50);
                    }, 500);

                return result;
            }
        }

        // Return the last keydown result by default, resort to false if unknown
        return last_keydown_result[keysym] || false;
    };

    /**
     * Marks a key as released, firing the keyup event if registered.
     *
     * @param {number} keysym
     *     The keysym of the key to release.
     */
    this.release = function (keysym) {
        // Only release if pressed
        if (guac_keyboard.pressed[keysym]) {
            // Mark key as released
            delete guac_keyboard.pressed[keysym];
            delete implicitlyPressed[keysym];

            // Stop repeat
            window.clearTimeout(key_repeat_timeout);
            window.clearInterval(key_repeat_interval);

            // Send key event
            if (keysym !== null && guac_keyboard.onkeyup) guac_keyboard.onkeyup(keysym);
        }
    };

    /**
     * Presses and releases the keys necessary to type the given string of
     * text.
     *
     * @param {!string} str
     *     The string to type.
     */
    this.type = function type(str) {
        // Press/release the key corresponding to each character in the string
        for (var i = 0; i < str.length; i++) {
            // Determine keysym of current character
            var codepoint = str.codePointAt ? str.codePointAt(i) : str.charCodeAt(i);
            var keysym = keysym_from_charcode(codepoint);

            // Press and release key for current character
            guac_keyboard.press(keysym);
            guac_keyboard.release(keysym);
        }
    };

    /**
     * Resets the state of this keyboard, releasing all keys, and firing keyup
     * events for each released key.
     */
    this.reset = function () {
        // Release all pressed keys
        for (var keysym in guac_keyboard.pressed) guac_keyboard.release(parseInt(keysym));

        // Clear event log
        eventLog = [];
    };

    /**
     * Resynchronizes the remote state of the given modifier with its
     * corresponding local modifier state, as dictated by
     * {@link KeyEvent#modifiers} within the given key event, by pressing or
     * releasing keysyms.
     *
     * @private
     * @param {!string} modifier
     *     The name of the {@link Guacamole.Keyboard.ModifierState} property
     *     being updated.
     *
     * @param {!number[]} keysyms
     *     The keysyms which represent the modifier being updated.
     *
     * @param {!KeyEvent} keyEvent
     *     Guacamole's current best interpretation of the key event being
     *     processed.
     */
    var updateModifierState = function updateModifierState(modifier, keysyms, keyEvent) {
        var localState = keyEvent.modifiers[modifier];
        var remoteState = guac_keyboard.modifiers[modifier];

        var i;

        // Do not trust changes in modifier state for events directly involving
        // that modifier: (1) the flag may erroneously be cleared despite
        // another version of the same key still being held and (2) the change
        // in flag may be due to the current event being processed, thus
        // updating things here is at best redundant and at worst incorrect
        if (keysyms.indexOf(keyEvent.keysym) !== -1) return;

        // Release all related keys if modifier is implicitly released
        if (remoteState && localState === false) {
            for (i = 0; i < keysyms.length; i++) {
                guac_keyboard.release(keysyms[i]);
            }
        }

        // Press if modifier is implicitly pressed
        else if (!remoteState && localState) {
            // Verify that modifier flag isn't already pressed or already set
            // due to another version of the same key being held down
            for (i = 0; i < keysyms.length; i++) {
                if (guac_keyboard.pressed[keysyms[i]]) return;
            }

            // Mark as implicitly pressed only if there is other information
            // within the key event relating to a different key. Some
            // platforms, such as iOS, will send essentially empty key events
            // for modifier keys, using only the modifier flags to signal the
            // identity of the key.
            var keysym = keysyms[0];
            if (keyEvent.keysym) implicitlyPressed[keysym] = true;

            guac_keyboard.press(keysym);
        }
    };

    /**
     * Given a keyboard event, updates the remote key state to match the local
     * modifier state and remote based on the modifier flags within the event.
     * This function pays no attention to keycodes.
     *
     * @private
     * @param {!KeyEvent} keyEvent
     *     Guacamole's current best interpretation of the key event being
     *     processed.
     */
    var syncModifierStates = function syncModifierStates(keyEvent) {
        // Resync state of alt
        updateModifierState(
            'alt',
            [
                0xffe9, // Left alt
                0xffea, // Right alt
                0xfe03, // AltGr
            ],
            keyEvent
        );

        // Resync state of shift
        updateModifierState(
            'shift',
            [
                0xffe1, // Left shift
                0xffe2, // Right shift
            ],
            keyEvent
        );

        // Resync state of ctrl
        updateModifierState(
            'ctrl',
            [
                0xffe3, // Left ctrl
                0xffe4, // Right ctrl
            ],
            keyEvent
        );

        // Resync state of meta
        updateModifierState(
            'meta',
            [
                0xffe7, // Left meta
                0xffe8, // Right meta
            ],
            keyEvent
        );

        // Resync state of hyper
        updateModifierState(
            'hyper',
            [
                0xffeb, // Left super/hyper
                0xffec, // Right super/hyper
            ],
            keyEvent
        );

        // Update state
        guac_keyboard.modifiers = keyEvent.modifiers;
    };

    /**
     * Returns whether all currently pressed keys were implicitly pressed. A
     * key is implicitly pressed if its status was inferred indirectly from
     * inspection of other key events.
     *
     * @private
     * @returns {!boolean}
     *     true if all currently pressed keys were implicitly pressed, false
     *     otherwise.
     */
    var isStateImplicit = function isStateImplicit() {
        for (var keysym in guac_keyboard.pressed) {
            if (!implicitlyPressed[keysym]) return false;
        }

        return true;
    };

    /**
     * Reads through the event log, removing events from the head of the log
     * when the corresponding true key presses are known (or as known as they
     * can be).
     *
     * @private
     * @return {boolean}
     *     Whether the default action of the latest event should be prevented.
     */
    function interpret_events() {
        // Do not prevent default if no event could be interpreted
        var handled_event = interpret_event();
        if (!handled_event) return false;

        // Interpret as much as possible
        var last_event;
        do {
            last_event = handled_event;
            handled_event = interpret_event();
        } while (handled_event !== null);

        // Reset keyboard state if we cannot expect to receive any further
        // keyup events
        if (isStateImplicit()) guac_keyboard.reset();

        return last_event.defaultPrevented;
    }

    /**
     * Releases Ctrl+Alt, if both are currently pressed and the given keysym
     * looks like a key that may require AltGr.
     *
     * @private
     * @param {!number} keysym
     *     The key that was just pressed.
     */
    var release_simulated_altgr = function release_simulated_altgr(keysym) {
        // Both Ctrl+Alt must be pressed if simulated AltGr is in use
        if (!guac_keyboard.modifiers.ctrl || !guac_keyboard.modifiers.alt) return;

        // Assume [A-Z] never require AltGr
        if (keysym >= 0x0041 && keysym <= 0x005a) return;

        // Assume [a-z] never require AltGr
        if (keysym >= 0x0061 && keysym <= 0x007a) return;

        // Release Ctrl+Alt if the keysym is printable
        if (keysym <= 0xff || (keysym & 0xff000000) === 0x01000000) {
            guac_keyboard.release(0xffe3); // Left ctrl
            guac_keyboard.release(0xffe4); // Right ctrl
            guac_keyboard.release(0xffe9); // Left alt
            guac_keyboard.release(0xffea); // Right alt
        }
    };

    /**
     * Reads through the event log, interpreting the first event, if possible,
     * and returning that event. If no events can be interpreted, due to a
     * total lack of events or the need for more events, null is returned. Any
     * interpreted events are automatically removed from the log.
     *
     * @private
     * @return {KeyEvent}
     *     The first key event in the log, if it can be interpreted, or null
     *     otherwise.
     */
    var interpret_event = function interpret_event() {
        // Peek at first event in log
        var first = eventLog[0];
        if (!first) return null;

        // Keydown event
        if (first instanceof KeydownEvent) {
            var keysym = null;
            var accepted_events = [];

            // Defer handling of Meta until it is known to be functioning as a
            // modifier (it may otherwise actually be an alternative method for
            // pressing a single key, such as Meta+Left for Home on ChromeOS)
            if (first.keysym === 0xffe7 || first.keysym === 0xffe8) {
                // Defer handling until further events exist to provide context
                if (eventLog.length === 1) return null;

                // Drop keydown if it turns out Meta does not actually apply
                if (eventLog[1].keysym !== first.keysym) {
                    if (!eventLog[1].modifiers.meta) return eventLog.shift();
                }

                // Drop duplicate keydown events while waiting to determine
                // whether to acknowledge Meta (browser may repeat keydown
                // while the key is held)
                else if (eventLog[1] instanceof KeydownEvent) return eventLog.shift();
            }

            // If event itself is reliable, no need to wait for other events
            if (first.reliable) {
                keysym = first.keysym;
                accepted_events = eventLog.splice(0, 1);
            }

            // If keydown is immediately followed by a keypress, use the indicated character
            else if (eventLog[1] instanceof KeypressEvent) {
                keysym = eventLog[1].keysym;
                accepted_events = eventLog.splice(0, 2);
            }

            // If keydown is immediately followed by anything else, then no
            // keypress can possibly occur to clarify this event, and we must
            // handle it now
            else if (eventLog[1]) {
                keysym = first.keysym;
                accepted_events = eventLog.splice(0, 1);
            }

            // Fire a key press if valid events were found
            if (accepted_events.length > 0) {
                syncModifierStates(first);

                if (keysym) {
                    // Fire event
                    release_simulated_altgr(keysym);
                    var defaultPrevented = !guac_keyboard.press(keysym);
                    recentKeysym[first.keyCode] = keysym;

                    // Release the key now if we cannot rely on the associated
                    // keyup event
                    if (!first.keyupReliable) guac_keyboard.release(keysym);

                    // Record whether default was prevented
                    for (var i = 0; i < accepted_events.length; i++) accepted_events[i].defaultPrevented = defaultPrevented;
                }

                return first;
            }
        } // end if keydown

        // Keyup event
        else if (first instanceof KeyupEvent && !quirks.keyupUnreliable) {
            // Release specific key if known
            let keysym = first.keysym;
            if (keysym) {
                guac_keyboard.release(keysym);
                delete recentKeysym[first.keyCode];
                first.defaultPrevented = true;
            }

            // Otherwise, fall back to releasing all keys
            else {
                guac_keyboard.reset();
                return first;
            }

            syncModifierStates(first);
            return eventLog.shift();
        } // end if keyup

        // Ignore any other type of event (keypress by itself is invalid, and
        // unreliable keyup events should simply be dumped)
        else return eventLog.shift();

        // No event interpreted
        return null;
    };

    /**
     * Returns the keyboard location of the key associated with the given
     * keyboard event. The location differentiates key events which otherwise
     * have the same keycode, such as left shift vs. right shift.
     *
     * @private
     * @param {!KeyboardEvent} e
     *     A JavaScript keyboard event, as received through the DOM via a
     *     "keydown", "keyup", or "keypress" handler.
     *
     * @returns {!number}
     *     The location of the key event on the keyboard, as defined at:
     *     http://www.w3.org/TR/DOM-Level-3-Events/#events-KeyboardEvent
     */
    var getEventLocation = function getEventLocation(e) {
        // Use standard location, if possible
        if ('location' in e) return e.location;

        // Failing that, attempt to use deprecated keyLocation
        if ('keyLocation' in e) return e.keyLocation;

        // If no location is available, assume left side
        return 0;
    };

    /**
     * Attempts to mark the given Event as having been handled by this
     * Guacamole.Keyboard. If the Event has already been marked as handled,
     * false is returned.
     *
     * @param {!Event} e
     *     The Event to mark.
     *
     * @returns {!boolean}
     *     true if the given Event was successfully marked, false if the given
     *     Event was already marked.
     */
    var markEvent = function markEvent(e) {
        // Fail if event is already marked
        if (e[EVENT_MARKER]) return false;

        // Mark event otherwise
        e[EVENT_MARKER] = true;
        return true;
    };

    /**
     * Attaches event listeners to the given Element, automatically translating
     * received key, input, and composition events into simple keydown/keyup
     * events signalled through this Guacamole.Keyboard's onkeydown and
     * onkeyup handlers.
     *
     * @param {!(Element|Document)} element
     *     The Element to attach event listeners to for the sake of handling
     *     key or input events.
     */
    this.listenTo = function listenTo(element) {
        // When key pressed
        element.addEventListener(
            'keydown',
            function (e) {
                // Only intercept if handler set
                if (!guac_keyboard.onkeydown) return;

                // Ignore events which have already been handled
                if (!markEvent(e)) return;

                var keydownEvent = new KeydownEvent(e);

                // Ignore (but do not prevent) the "composition" keycode sent by some
                // browsers when an IME is in use (see: http://lists.w3.org/Archives/Public/www-dom/2010JulSep/att-0182/keyCode-spec.html)
                if (keydownEvent.keyCode === 229) return;

                // Log event
                eventLog.push(keydownEvent);

                // Interpret as many events as possible, prevent default if indicated
                if (interpret_events()) e.preventDefault();
            },
            true
        );

        // When key pressed
        element.addEventListener(
            'keypress',
            function (e) {
                // Only intercept if handler set
                if (!guac_keyboard.onkeydown && !guac_keyboard.onkeyup) return;

                // Ignore events which have already been handled
                if (!markEvent(e)) return;

                // Log event
                eventLog.push(new KeypressEvent(e));

                // Interpret as many events as possible, prevent default if indicated
                if (interpret_events()) e.preventDefault();
            },
            true
        );

        // When key released
        element.addEventListener(
            'keyup',
            function (e) {
                // Only intercept if handler set
                if (!guac_keyboard.onkeyup) return;

                // Ignore events which have already been handled
                if (!markEvent(e)) return;

                e.preventDefault();

                // Log event, call for interpretation
                eventLog.push(new KeyupEvent(e));
                interpret_events();
            },
            true
        );

        /**
         * Handles the given "input" event, typing the data within the input text.
         * If the event is complete (text is provided), handling of "compositionend"
         * events is suspended, as such events may conflict with input events.
         *
         * @private
         * @param {!InputEvent} e
         *     The "input" event to handle.
         */
        var handleInput = function handleInput(e) {
            // Only intercept if handler set
            if (!guac_keyboard.onkeydown && !guac_keyboard.onkeyup) return;

            // Ignore events which have already been handled
            if (!markEvent(e)) return;

            // Type all content written
            if (e.data && !e.isComposing) {
                element.removeEventListener('compositionend', handleComposition, false);
                guac_keyboard.type(e.data);
            }
        };

        /**
         * Handles the given "compositionend" event, typing the data within the
         * composed text. If the event is complete (composed text is provided),
         * handling of "input" events is suspended, as such events may conflict
         * with composition events.
         *
         * @private
         * @param {!CompositionEvent} e
         *     The "compositionend" event to handle.
         */
        var handleComposition = function handleComposition(e) {
            // Only intercept if handler set
            if (!guac_keyboard.onkeydown && !guac_keyboard.onkeyup) return;

            // Ignore events which have already been handled
            if (!markEvent(e)) return;

            // Type all content written
            if (e.data) {
                element.removeEventListener('input', handleInput, false);
                guac_keyboard.type(e.data);
            }
        };

        // Automatically type text entered into the wrapped field
        element.addEventListener('input', handleInput, false);
        element.addEventListener('compositionend', handleComposition, false);
    };

    // Listen to given element, if any
    if (element) guac_keyboard.listenTo(element);
};

/**
 * The unique numerical identifier to assign to the next Guacamole.Keyboard
 * instance.
 *
 * @private
 * @type {!number}
 */
Guacamole.Keyboard._nextID = 0;

/**
 * The state of all supported keyboard modifiers.
 * @constructor
 */
Guacamole.Keyboard.ModifierState = function () {
    /**
     * Whether shift is currently pressed.
     *
     * @type {!boolean}
     */
    this.shift = false;

    /**
     * Whether ctrl is currently pressed.
     *
     * @type {!boolean}
     */
    this.ctrl = false;

    /**
     * Whether alt is currently pressed.
     *
     * @type {!boolean}
     */
    this.alt = false;

    /**
     * Whether meta (apple key) is currently pressed.
     *
     * @type {!boolean}
     */
    this.meta = false;

    /**
     * Whether hyper (windows key) is currently pressed.
     *
     * @type {!boolean}
     */
    this.hyper = false;
};

/**
 * Returns the modifier state applicable to the keyboard event given.
 *
 * @param {!KeyboardEvent} e
 *     The keyboard event to read.
 *
 * @returns {!Guacamole.Keyboard.ModifierState}
 *     The current state of keyboard modifiers.
 */
Guacamole.Keyboard.ModifierState.fromKeyboardEvent = function (e) {
    var state = new Guacamole.Keyboard.ModifierState();

    // Assign states from old flags
    state.shift = e.shiftKey;
    state.ctrl = e.ctrlKey;
    state.alt = e.altKey;
    state.meta = e.metaKey;

    // Use DOM3 getModifierState() for others
    if (e.getModifierState) {
        state.hyper = e.getModifierState('OS') || e.getModifierState('Super') || e.getModifierState('Hyper') || e.getModifierState('Win');
    }

    return state;
};

/**
 * Abstract ordered drawing surface. Each Layer contains a canvas element and
 * provides simple drawing instructions for drawing to that canvas element,
 * however unlike the canvas element itself, drawing operations on a Layer are
 * guaranteed to run in order, even if such an operation must wait for an image
 * to load before completing.
 *
 * @constructor
 *
 * @param {!number} width
 *     The width of the Layer, in pixels. The canvas element backing this Layer
 *     will be given this width.
 *
 * @param {!number} height
 *     The height of the Layer, in pixels. The canvas element backing this
 *     Layer will be given this height.
 */
Guacamole.Layer = function (width, height) {
    /**
     * Reference to this Layer.
     *
     * @private
     * @type {!Guacamole.Layer}
     */
    var layer = this;

    /**
     * The number of pixels the width or height of a layer must change before
     * the underlying canvas is resized. The underlying canvas will be kept at
     * dimensions which are integer multiples of this factor.
     *
     * @private
     * @constant
     * @type {!number}
     */
    var CANVAS_SIZE_FACTOR = 64;

    /**
     * The canvas element backing this Layer.
     *
     * @private
     * @type {!HTMLCanvasElement}
     */
    var canvas = document.createElement('canvas');

    /**
     * The 2D display context of the canvas element backing this Layer.
     *
     * @private
     * @type {!CanvasRenderingContext2D}
     */
    var context = canvas.getContext('2d');
    context.save();

    /**
     * Whether the layer has not yet been drawn to. Once any draw operation
     * which affects the underlying canvas is invoked, this flag will be set to
     * false.
     *
     * @private
     * @type {!boolean}
     */
    var empty = true;

    /**
     * Whether a new path should be started with the next path drawing
     * operations.
     *
     * @private
     * @type {!boolean}
     */
    var pathClosed = true;

    /**
     * The number of states on the state stack.
     *
     * Note that there will ALWAYS be one element on the stack, but that
     * element is not exposed. It is only used to reset the layer to its
     * initial state.
     *
     * @private
     * @type {!number}
     */
    var stackSize = 0;

    /**
     * Map of all Guacamole channel masks to HTML5 canvas composite operation
     * names. Not all channel mask combinations are currently implemented.
     *
     * @private
     * @type {!Object.<number, string>}
     */
    var compositeOperation = {
        /* 0x0 NOT IMPLEMENTED */
        0x1: 'destination-in',
        0x2: 'destination-out',
        /* 0x3 NOT IMPLEMENTED */
        0x4: 'source-in',
        /* 0x5 NOT IMPLEMENTED */
        0x6: 'source-atop',
        /* 0x7 NOT IMPLEMENTED */
        0x8: 'source-out',
        0x9: 'destination-atop',
        0xa: 'xor',
        0xb: 'destination-over',
        0xc: 'copy',
        /* 0xD NOT IMPLEMENTED */
        0xe: 'source-over',
        0xf: 'lighter',
    };

    /**
     * Resizes the canvas element backing this Layer. This function should only
     * be used internally.
     *
     * @private
     * @param {number} [newWidth=0]
     *     The new width to assign to this Layer.
     *
     * @param {number} [newHeight=0]
     *     The new height to assign to this Layer.
     */
    var resize = function resize(newWidth, newHeight) {
        // Default size to zero
        newWidth = newWidth || 0;
        newHeight = newHeight || 0;

        // Calculate new dimensions of internal canvas
        var canvasWidth = Math.ceil(newWidth / CANVAS_SIZE_FACTOR) * CANVAS_SIZE_FACTOR;
        var canvasHeight = Math.ceil(newHeight / CANVAS_SIZE_FACTOR) * CANVAS_SIZE_FACTOR;

        // Resize only if canvas dimensions are actually changing
        if (canvas.width !== canvasWidth || canvas.height !== canvasHeight) {
            // Copy old data only if relevant and non-empty
            var oldData = null;
            if (!empty && canvas.width !== 0 && canvas.height !== 0) {
                // Create canvas and context for holding old data
                oldData = document.createElement('canvas');
                oldData.width = Math.min(layer.width, newWidth);
                oldData.height = Math.min(layer.height, newHeight);

                var oldDataContext = oldData.getContext('2d');

                // Copy image data from current
                oldDataContext.drawImage(canvas, 0, 0, oldData.width, oldData.height, 0, 0, oldData.width, oldData.height);
            }

            // Preserve composite operation
            var oldCompositeOperation = context.globalCompositeOperation;

            // Resize canvas
            canvas.width = canvasWidth;
            canvas.height = canvasHeight;

            // Redraw old data, if any
            if (oldData) context.drawImage(oldData, 0, 0, oldData.width, oldData.height, 0, 0, oldData.width, oldData.height);

            // Restore composite operation
            context.globalCompositeOperation = oldCompositeOperation;

            // Acknowledge reset of stack (happens on resize of canvas)
            stackSize = 0;
            context.save();
        }

        // If the canvas size is not changing, manually force state reset
        else layer.reset();

        // Assign new layer dimensions
        layer.width = newWidth;
        layer.height = newHeight;
    };

    /**
     * Given the X and Y coordinates of the upper-left corner of a rectangle
     * and the rectangle's width and height, resize the backing canvas element
     * as necessary to ensure that the rectangle fits within the canvas
     * element's coordinate space. This function will only make the canvas
     * larger. If the rectangle already fits within the canvas element's
     * coordinate space, the canvas is left unchanged.
     *
     * @private
     * @param {!number} x
     *     The X coordinate of the upper-left corner of the rectangle to fit.
     *
     * @param {!number} y
     *     The Y coordinate of the upper-left corner of the rectangle to fit.
     *
     * @param {!number} w
     *     The width of the rectangle to fit.
     *
     * @param {!number} h
     *     The height of the rectangle to fit.
     */
    function fitRect(x, y, w, h) {
        // Calculate bounds
        var opBoundX = w + x;
        var opBoundY = h + y;

        // Determine max width
        var resizeWidth;
        if (opBoundX > layer.width) resizeWidth = opBoundX;
        else resizeWidth = layer.width;

        // Determine max height
        var resizeHeight;
        if (opBoundY > layer.height) resizeHeight = opBoundY;
        else resizeHeight = layer.height;

        // Resize if necessary
        layer.resize(resizeWidth, resizeHeight);
    }

    /**
     * Set to true if this Layer should resize itself to accommodate the
     * dimensions of any drawing operation, and false (the default) otherwise.
     *
     * Note that setting this property takes effect immediately, and thus may
     * take effect on operations that were started in the past but have not
     * yet completed. If you wish the setting of this flag to only modify
     * future operations, you will need to make the setting of this flag an
     * operation with sync().
     *
     * @example
     * // Set autosize to true for all future operations
     * layer.sync(function() {
     *     layer.autosize = true;
     * });
     *
     * @type {!boolean}
     * @default false
     */
    this.autosize = false;

    /**
     * The current width of this layer.
     *
     * @type {!number}
     */
    this.width = width;

    /**
     * The current height of this layer.
     *
     * @type {!number}
     */
    this.height = height;

    /**
     * Returns the canvas element backing this Layer. Note that the dimensions
     * of the canvas may not exactly match those of the Layer, as resizing a
     * canvas while maintaining its state is an expensive operation.
     *
     * @returns {!HTMLCanvasElement}
     *     The canvas element backing this Layer.
     */
    this.getCanvas = function getCanvas() {
        return canvas;
    };

    /**
     * Returns a new canvas element containing the same image as this Layer.
     * Unlike getCanvas(), the canvas element returned is guaranteed to have
     * the exact same dimensions as the Layer.
     *
     * @returns {!HTMLCanvasElement}
     *     A new canvas element containing a copy of the image content this
     *     Layer.
     */
    this.toCanvas = function toCanvas() {
        // Create new canvas having same dimensions
        var canvas = document.createElement('canvas');
        canvas.width = layer.width;
        canvas.height = layer.height;

        // Copy image contents to new canvas
        var context = canvas.getContext('2d');
        context.drawImage(layer.getCanvas(), 0, 0);

        return canvas;
    };

    /**
     * Changes the size of this Layer to the given width and height. Resizing
     * is only attempted if the new size provided is actually different from
     * the current size.
     *
     * @param {!number} newWidth
     *     The new width to assign to this Layer.
     *
     * @param {!number} newHeight
     *     The new height to assign to this Layer.
     */
    this.resize = function (newWidth, newHeight) {
        if (newWidth !== layer.width || newHeight !== layer.height) resize(newWidth, newHeight);
    };

    /**
     * Draws the specified image at the given coordinates. The image specified
     * must already be loaded.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!CanvasImageSource} image
     *     The image to draw. Note that this is not a URL.
     */
    this.drawImage = function (x, y, image) {
        if (layer.autosize) fitRect(x, y, image.width, image.height);
        context.drawImage(image, x, y);
        empty = false;
    };

    /**
     * Transfer a rectangle of image data from one Layer to this Layer using the
     * specified transfer function.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The Layer to copy image data from.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     *
     * @param {!function} transferFunction
     *     The transfer function to use to transfer data from source to
     *     destination.
     */
    this.transfer = function (srcLayer, srcx, srcy, srcw, srch, x, y, transferFunction) {
        var srcCanvas = srcLayer.getCanvas();

        // If entire rectangle outside source canvas, stop
        if (srcx >= srcCanvas.width || srcy >= srcCanvas.height) return;

        // Otherwise, clip rectangle to area
        if (srcx + srcw > srcCanvas.width) srcw = srcCanvas.width - srcx;

        if (srcy + srch > srcCanvas.height) srch = srcCanvas.height - srcy;

        // Stop if nothing to draw.
        if (srcw === 0 || srch === 0) return;

        if (layer.autosize) fitRect(x, y, srcw, srch);

        // Get image data from src and dst
        var src = srcLayer.getCanvas().getContext('2d').getImageData(srcx, srcy, srcw, srch);
        var dst = context.getImageData(x, y, srcw, srch);

        // Apply transfer for each pixel
        for (var i = 0; i < srcw * srch * 4; i += 4) {
            // Get source pixel environment
            var src_pixel = new Guacamole.Layer.Pixel(src.data[i], src.data[i + 1], src.data[i + 2], src.data[i + 3]);

            // Get destination pixel environment
            var dst_pixel = new Guacamole.Layer.Pixel(dst.data[i], dst.data[i + 1], dst.data[i + 2], dst.data[i + 3]);

            // Apply transfer function
            transferFunction(src_pixel, dst_pixel);

            // Save pixel data
            dst.data[i] = dst_pixel.red;
            dst.data[i + 1] = dst_pixel.green;
            dst.data[i + 2] = dst_pixel.blue;
            dst.data[i + 3] = dst_pixel.alpha;
        }

        // Draw image data
        context.putImageData(dst, x, y);
        empty = false;
    };

    /**
     * Put a rectangle of image data from one Layer to this Layer directly
     * without performing any alpha blending. Simply copy the data.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The Layer to copy image data from.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     */
    this.put = function (srcLayer, srcx, srcy, srcw, srch, x, y) {
        var srcCanvas = srcLayer.getCanvas();

        // If entire rectangle outside source canvas, stop
        if (srcx >= srcCanvas.width || srcy >= srcCanvas.height) return;

        // Otherwise, clip rectangle to area
        if (srcx + srcw > srcCanvas.width) srcw = srcCanvas.width - srcx;

        if (srcy + srch > srcCanvas.height) srch = srcCanvas.height - srcy;

        // Stop if nothing to draw.
        if (srcw === 0 || srch === 0) return;

        if (layer.autosize) fitRect(x, y, srcw, srch);

        // Get image data from src and dst
        var src = srcLayer.getCanvas().getContext('2d').getImageData(srcx, srcy, srcw, srch);
        context.putImageData(src, x, y);
        empty = false;
    };

    /**
     * Copy a rectangle of image data from one Layer to this Layer. This
     * operation will copy exactly the image data that will be drawn once all
     * operations of the source Layer that were pending at the time this
     * function was called are complete. This operation will not alter the
     * size of the source Layer even if its autosize property is set to true.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The Layer to copy image data from.
     *
     * @param {!number} srcx
     *     The X coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcy
     *     The Y coordinate of the upper-left corner of the rectangle within
     *     the source Layer's coordinate space to copy data from.
     *
     * @param {!number} srcw
     *     The width of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} srch
     *     The height of the rectangle within the source Layer's coordinate
     *     space to copy data from.
     *
     * @param {!number} x
     *     The destination X coordinate.
     *
     * @param {!number} y
     *     The destination Y coordinate.
     */
    this.copy = function (srcLayer, srcx, srcy, srcw, srch, x, y) {
        var srcCanvas = srcLayer.getCanvas();

        // If entire rectangle outside source canvas, stop
        if (srcx >= srcCanvas.width || srcy >= srcCanvas.height) return;

        // Otherwise, clip rectangle to area
        if (srcx + srcw > srcCanvas.width) srcw = srcCanvas.width - srcx;

        if (srcy + srch > srcCanvas.height) srch = srcCanvas.height - srcy;

        // Stop if nothing to draw.
        if (srcw === 0 || srch === 0) return;

        if (layer.autosize) fitRect(x, y, srcw, srch);
        context.drawImage(srcCanvas, srcx, srcy, srcw, srch, x, y, srcw, srch);
        empty = false;
    };

    /**
     * Starts a new path at the specified point.
     *
     * @param {!number} x
     *     The X coordinate of the point to draw.
     *
     * @param {!number} y
     *     The Y coordinate of the point to draw.
     */
    this.moveTo = function (x, y) {
        // Start a new path if current path is closed
        if (pathClosed) {
            context.beginPath();
            pathClosed = false;
        }

        if (layer.autosize) fitRect(x, y, 0, 0);
        context.moveTo(x, y);
    };

    /**
     * Add the specified line to the current path.
     *
     * @param {!number} x
     *     The X coordinate of the endpoint of the line to draw.
     *
     * @param {!number} y
     *     The Y coordinate of the endpoint of the line to draw.
     */
    this.lineTo = function (x, y) {
        // Start a new path if current path is closed
        if (pathClosed) {
            context.beginPath();
            pathClosed = false;
        }

        if (layer.autosize) fitRect(x, y, 0, 0);
        context.lineTo(x, y);
    };

    /**
     * Add the specified arc to the current path.
     *
     * @param {!number} x
     *     The X coordinate of the center of the circle which will contain the
     *     arc.
     *
     * @param {!number} y
     *     The Y coordinate of the center of the circle which will contain the
     *     arc.
     *
     * @param {!number} radius
     *     The radius of the circle.
     *
     * @param {!number} startAngle
     *     The starting angle of the arc, in radians.
     *
     * @param {!number} endAngle
     *     The ending angle of the arc, in radians.
     *
     * @param {!boolean} negative
     *     Whether the arc should be drawn in order of decreasing angle.
     */
    this.arc = function (x, y, radius, startAngle, endAngle, negative) {
        // Start a new path if current path is closed
        if (pathClosed) {
            context.beginPath();
            pathClosed = false;
        }

        if (layer.autosize) fitRect(x, y, 0, 0);
        context.arc(x, y, radius, startAngle, endAngle, negative);
    };

    /**
     * Starts a new path at the specified point.
     *
     * @param {!number} cp1x
     *     The X coordinate of the first control point.
     *
     * @param {!number} cp1y
     *     The Y coordinate of the first control point.
     *
     * @param {!number} cp2x
     *     The X coordinate of the second control point.
     *
     * @param {!number} cp2y
     *     The Y coordinate of the second control point.
     *
     * @param {!number} x
     *     The X coordinate of the endpoint of the curve.
     *
     * @param {!number} y
     *     The Y coordinate of the endpoint of the curve.
     */
    this.curveTo = function (cp1x, cp1y, cp2x, cp2y, x, y) {
        // Start a new path if current path is closed
        if (pathClosed) {
            context.beginPath();
            pathClosed = false;
        }

        if (layer.autosize) fitRect(x, y, 0, 0);
        context.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, x, y);
    };

    /**
     * Closes the current path by connecting the end point with the start
     * point (if any) with a straight line.
     */
    this.close = function () {
        context.closePath();
        pathClosed = true;
    };

    /**
     * Add the specified rectangle to the current path.
     *
     * @param {!number} x
     *     The X coordinate of the upper-left corner of the rectangle to draw.
     *
     * @param {!number} y
     *     The Y coordinate of the upper-left corner of the rectangle to draw.
     *
     * @param {!number} w
     *     The width of the rectangle to draw.
     *
     * @param {!number} h
     *     The height of the rectangle to draw.
     */
    this.rect = function (x, y, w, h) {
        // Start a new path if current path is closed
        if (pathClosed) {
            context.beginPath();
            pathClosed = false;
        }

        if (layer.autosize) fitRect(x, y, w, h);
        context.rect(x, y, w, h);
    };

    /**
     * Clip all future drawing operations by the current path. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as fillColor()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     */
    this.clip = function () {
        // Set new clipping region
        context.clip();

        // Path now implicitly closed
        pathClosed = true;
    };

    /**
     * Stroke the current path with the specified color. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!string} cap
     *     The line cap style. Can be "round", "square", or "butt".
     *
     * @param {!string} join
     *     The line join style. Can be "round", "bevel", or "miter".
     *
     * @param {!number} thickness
     *     The line thickness in pixels.
     *
     * @param {!number} r
     *     The red component of the color to fill.
     *
     * @param {!number} g
     *     The green component of the color to fill.
     *
     * @param {!number} b
     *     The blue component of the color to fill.
     *
     * @param {!number} a
     *     The alpha component of the color to fill.
     */
    this.strokeColor = function (cap, join, thickness, r, g, b, a) {
        // Stroke with color
        context.lineCap = cap;
        context.lineJoin = join;
        context.lineWidth = thickness;
        context.strokeStyle = 'rgba(' + r + ',' + g + ',' + b + ',' + a / 255.0 + ')';
        context.stroke();
        empty = false;

        // Path now implicitly closed
        pathClosed = true;
    };

    /**
     * Fills the current path with the specified color. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!number} r
     *     The red component of the color to fill.
     *
     * @param {!number} g
     *     The green component of the color to fill.
     *
     * @param {!number} b
     *     The blue component of the color to fill.
     *
     * @param {!number} a
     *     The alpha component of the color to fill.
     */
    this.fillColor = function (r, g, b, a) {
        // Fill with color
        context.fillStyle = 'rgba(' + r + ',' + g + ',' + b + ',' + a / 255.0 + ')';
        context.fill();
        empty = false;

        // Path now implicitly closed
        pathClosed = true;
    };

    /**
     * Stroke the current path with the image within the specified layer. The
     * image data will be tiled infinitely within the stroke. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!string} cap
     *     The line cap style. Can be "round", "square", or "butt".
     *
     * @param {!string} join
     *     The line join style. Can be "round", "bevel", or "miter".
     *
     * @param {!number} thickness
     *     The line thickness in pixels.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The layer to use as a repeating pattern within the stroke.
     */
    this.strokeLayer = function (cap, join, thickness, srcLayer) {
        // Stroke with image data
        context.lineCap = cap;
        context.lineJoin = join;
        context.lineWidth = thickness;
        context.strokeStyle = context.createPattern(srcLayer.getCanvas(), 'repeat');
        context.stroke();
        empty = false;

        // Path now implicitly closed
        pathClosed = true;
    };

    /**
     * Fills the current path with the image within the specified layer. The
     * image data will be tiled infinitely within the stroke. The current path
     * is implicitly closed. The current path can continue to be reused
     * for other operations (such as clip()) but a new path will be started
     * once a path drawing operation (path() or rect()) is used.
     *
     * @param {!Guacamole.Layer} srcLayer
     *     The layer to use as a repeating pattern within the fill.
     */
    this.fillLayer = function (srcLayer) {
        // Fill with image data
        context.fillStyle = context.createPattern(srcLayer.getCanvas(), 'repeat');
        context.fill();
        empty = false;

        // Path now implicitly closed
        pathClosed = true;
    };

    /**
     * Push current layer state onto stack.
     */
    this.push = function () {
        // Save current state onto stack
        context.save();
        stackSize++;
    };

    /**
     * Pop layer state off stack.
     */
    this.pop = function () {
        // Restore current state from stack
        if (stackSize > 0) {
            context.restore();
            stackSize--;
        }
    };

    /**
     * Reset the layer, clearing the stack, the current path, and any transform
     * matrix.
     */
    this.reset = function () {
        // Clear stack
        while (stackSize > 0) {
            context.restore();
            stackSize--;
        }

        // Restore to initial state
        context.restore();
        context.save();

        // Clear path
        context.beginPath();
        pathClosed = false;
    };

    /**
     * Sets the given affine transform (defined with six values from the
     * transform's matrix).
     *
     * @param {!number} a
     *     The first value in the affine transform's matrix.
     *
     * @param {!number} b
     *     The second value in the affine transform's matrix.
     *
     * @param {!number} c
     *     The third value in the affine transform's matrix.
     *
     * @param {!number} d
     *     The fourth value in the affine transform's matrix.
     *
     * @param {!number} e
     *     The fifth value in the affine transform's matrix.
     *
     * @param {!number} f
     *     The sixth value in the affine transform's matrix.
     */
    this.setTransform = function (a, b, c, d, e, f) {
        context.setTransform(
            a,
            b,
            c,
            d,
            e,
            f
            /*0, 0, 1*/
        );
    };

    /**
     * Applies the given affine transform (defined with six values from the
     * transform's matrix).
     *
     * @param {!number} a
     *     The first value in the affine transform's matrix.
     *
     * @param {!number} b
     *     The second value in the affine transform's matrix.
     *
     * @param {!number} c
     *     The third value in the affine transform's matrix.
     *
     * @param {!number} d
     *     The fourth value in the affine transform's matrix.
     *
     * @param {!number} e
     *     The fifth value in the affine transform's matrix.
     *
     * @param {!number} f
     *     The sixth value in the affine transform's matrix.
     */
    this.transform = function (a, b, c, d, e, f) {
        context.transform(
            a,
            b,
            c,
            d,
            e,
            f
            /*0, 0, 1*/
        );
    };

    /**
     * Sets the channel mask for future operations on this Layer.
     *
     * The channel mask is a Guacamole-specific compositing operation identifier
     * with a single bit representing each of four channels (in order): source
     * image where destination transparent, source where destination opaque,
     * destination where source transparent, and destination where source
     * opaque.
     *
     * @param {!number} mask
     *     The channel mask for future operations on this Layer.
     */
    this.setChannelMask = function (mask) {
        context.globalCompositeOperation = compositeOperation[mask];
    };

    /**
     * Sets the miter limit for stroke operations using the miter join. This
     * limit is the maximum ratio of the size of the miter join to the stroke
     * width. If this ratio is exceeded, the miter will not be drawn for that
     * joint of the path.
     *
     * @param {!number} limit
     *     The miter limit for stroke operations using the miter join.
     */
    this.setMiterLimit = function (limit) {
        context.miterLimit = limit;
    };

    // Initialize canvas dimensions
    resize(width, height);

    // Explicitly render canvas below other elements in the layer (such as
    // child layers). Chrome and others may fail to render layers properly
    // without this.
    canvas.style.zIndex = -1;
};

/**
 * Channel mask for the composite operation "rout".
 *
 * @type {!number}
 */
Guacamole.Layer.ROUT = 0x2;

/**
 * Channel mask for the composite operation "atop".
 *
 * @type {!number}
 */
Guacamole.Layer.ATOP = 0x6;

/**
 * Channel mask for the composite operation "xor".
 *
 * @type {!number}
 */
Guacamole.Layer.XOR = 0xa;

/**
 * Channel mask for the composite operation "rover".
 *
 * @type {!number}
 */
Guacamole.Layer.ROVER = 0xb;

/**
 * Channel mask for the composite operation "over".
 *
 * @type {!number}
 */
Guacamole.Layer.OVER = 0xe;

/**
 * Channel mask for the composite operation "plus".
 *
 * @type {!number}
 */
Guacamole.Layer.PLUS = 0xf;

/**
 * Channel mask for the composite operation "rin".
 * Beware that WebKit-based browsers may leave the contents of the destination
 * layer where the source layer is transparent, despite the definition of this
 * operation.
 *
 * @type {!number}
 */
Guacamole.Layer.RIN = 0x1;

/**
 * Channel mask for the composite operation "in".
 * Beware that WebKit-based browsers may leave the contents of the destination
 * layer where the source layer is transparent, despite the definition of this
 * operation.
 *
 * @type {!number}
 */
Guacamole.Layer.IN = 0x4;

/**
 * Channel mask for the composite operation "out".
 * Beware that WebKit-based browsers may leave the contents of the destination
 * layer where the source layer is transparent, despite the definition of this
 * operation.
 *
 * @type {!number}
 */
Guacamole.Layer.OUT = 0x8;

/**
 * Channel mask for the composite operation "ratop".
 * Beware that WebKit-based browsers may leave the contents of the destination
 * layer where the source layer is transparent, despite the definition of this
 * operation.
 *
 * @type {!number}
 */
Guacamole.Layer.RATOP = 0x9;

/**
 * Channel mask for the composite operation "src".
 * Beware that WebKit-based browsers may leave the contents of the destination
 * layer where the source layer is transparent, despite the definition of this
 * operation.
 *
 * @type {!number}
 */
Guacamole.Layer.SRC = 0xc;

/**
 * Represents a single pixel of image data. All components have a minimum value
 * of 0 and a maximum value of 255.
 *
 * @constructor
 *
 * @param {!number} r
 *     The red component of this pixel.
 *
 * @param {!number} g
 *     The green component of this pixel.
 *
 * @param {!number} b
 *     The blue component of this pixel.
 *
 * @param {!number} a
 *     The alpha component of this pixel.
 */
Guacamole.Layer.Pixel = function (r, g, b, a) {
    /**
     * The red component of this pixel, where 0 is the minimum value,
     * and 255 is the maximum.
     *
     * @type {!number}
     */
    this.red = r;

    /**
     * The green component of this pixel, where 0 is the minimum value,
     * and 255 is the maximum.
     *
     * @type {!number}
     */
    this.green = g;

    /**
     * The blue component of this pixel, where 0 is the minimum value,
     * and 255 is the maximum.
     *
     * @type {!number}
     */
    this.blue = b;

    /**
     * The alpha component of this pixel, where 0 is the minimum value,
     * and 255 is the maximum.
     *
     * @type {!number}
     */
    this.alpha = a;
};

/**
 * Provides cross-browser mouse events for a given element. The events of
 * the given element are automatically populated with handlers that translate
 * mouse events into a non-browser-specific event provided by the
 * Guacamole.Mouse instance.
 *
 * @example
 * var mouse = new Guacamole.Mouse(client.getDisplay().getElement());
 *
 * // Forward all mouse interaction over Guacamole connection
 * mouse.onEach(['mousedown', 'mousemove', 'mouseup'], function sendMouseEvent(e) {
 *     client.sendMouseState(e.state, true);
 * });
 *
 * @example
 * // Hide software cursor when mouse leaves display
 * mouse.on('mouseout', function hideCursor() {
 *     client.getDisplay().showCursor(false);
 * });
 *
 * @constructor
 * @augments Guacamole.Mouse.Event.Target
 * @param {!Element} element
 *     The Element to use to provide mouse events.
 */
Guacamole.Mouse = function Mouse(element) {
    Guacamole.Mouse.Event.Target.call(this);

    /**
     * Reference to this Guacamole.Mouse.
     *
     * @private
     * @type {!Guacamole.Mouse}
     */
    var guac_mouse = this;

    /**
     * The number of mousemove events to require before re-enabling mouse
     * event handling after receiving a touch event.
     *
     * @type {!number}
     */
    this.touchMouseThreshold = 3;

    /**
     * The minimum amount of pixels scrolled required for a single scroll button
     * click.
     *
     * @type {!number}
     */
    this.scrollThreshold = 53;

    /**
     * The number of pixels to scroll per line.
     *
     * @type {!number}
     */
    this.PIXELS_PER_LINE = 18;

    /**
     * The number of pixels to scroll per page.
     *
     * @type {!number}
     */
    this.PIXELS_PER_PAGE = this.PIXELS_PER_LINE * 16;

    /**
     * Array of {@link Guacamole.Mouse.State} button names corresponding to the
     * mouse button indices used by DOM mouse events.
     *
     * @private
     * @type {!string[]}
     */
    var MOUSE_BUTTONS = [Guacamole.Mouse.State.Buttons.LEFT, Guacamole.Mouse.State.Buttons.MIDDLE, Guacamole.Mouse.State.Buttons.RIGHT];

    /**
     * Counter of mouse events to ignore. This decremented by mousemove, and
     * while non-zero, mouse events will have no effect.
     *
     * @private
     * @type {!number}
     */
    var ignore_mouse = 0;

    /**
     * Cumulative scroll delta amount. This value is accumulated through scroll
     * events and results in scroll button clicks if it exceeds a certain
     * threshold.
     *
     * @private
     * @type {!number}
     */
    var scroll_delta = 0;

    // Block context menu so right-click gets sent properly
    element.addEventListener(
        'contextmenu',
        function (e) {
            Guacamole.Event.DOMEvent.cancelEvent(e);
        },
        false
    );

    element.addEventListener(
        'mousemove',
        function (e) {
            // If ignoring events, decrement counter
            if (ignore_mouse) {
                Guacamole.Event.DOMEvent.cancelEvent(e);
                ignore_mouse--;
                return;
            }

            guac_mouse.move(Guacamole.Position.fromClientPosition(element, e.clientX, e.clientY), e);
        },
        false
    );

    element.addEventListener(
        'mousedown',
        function (e) {
            // Do not handle if ignoring events
            if (ignore_mouse) {
                Guacamole.Event.DOMEvent.cancelEvent(e);
                return;
            }

            var button = MOUSE_BUTTONS[e.button];
            if (button) guac_mouse.press(button, e);
        },
        false
    );

    element.addEventListener(
        'mouseup',
        function (e) {
            // Do not handle if ignoring events
            if (ignore_mouse) {
                Guacamole.Event.DOMEvent.cancelEvent(e);
                return;
            }

            var button = MOUSE_BUTTONS[e.button];
            if (button) guac_mouse.release(button, e);
        },
        false
    );

    element.addEventListener(
        'mouseout',
        function (e) {
            // Get parent of the element the mouse pointer is leaving
            if (!e) e = window.event;

            // Check that mouseout is due to actually LEAVING the element
            var target = e.relatedTarget || e.toElement;
            while (target) {
                if (target === element) return;
                target = target.parentNode;
            }

            // Release all buttons and fire mouseout
            guac_mouse.reset(e);
            guac_mouse.out(e);
        },
        false
    );

    // Override selection on mouse event element.
    element.addEventListener(
        'selectstart',
        function (e) {
            Guacamole.Event.DOMEvent.cancelEvent(e);
        },
        false
    );

    // Ignore all pending mouse events when touch events are the apparent source
    function ignorePendingMouseEvents() {
        ignore_mouse = guac_mouse.touchMouseThreshold;
    }

    element.addEventListener('touchmove', ignorePendingMouseEvents, false);
    element.addEventListener('touchstart', ignorePendingMouseEvents, false);
    element.addEventListener('touchend', ignorePendingMouseEvents, false);

    // Scroll wheel support
    function mousewheel_handler(e) {
        // Determine approximate scroll amount (in pixels)
        var delta = e.deltaY || -e.wheelDeltaY || -e.wheelDelta;

        // If successfully retrieved scroll amount, convert to pixels if not
        // already in pixels
        if (delta) {
            // Convert to pixels if delta was lines
            if (e.deltaMode === 1) delta = e.deltaY * guac_mouse.PIXELS_PER_LINE;
            // Convert to pixels if delta was pages
            else if (e.deltaMode === 2) delta = e.deltaY * guac_mouse.PIXELS_PER_PAGE;
        }

        // Otherwise, assume legacy mousewheel event and line scrolling
        else delta = e.detail * guac_mouse.PIXELS_PER_LINE;

        // Update overall delta
        scroll_delta += delta;

        // Up
        if (scroll_delta <= -guac_mouse.scrollThreshold) {
            // Repeatedly click the up button until insufficient delta remains
            do {
                guac_mouse.click(Guacamole.Mouse.State.Buttons.UP);
                scroll_delta += guac_mouse.scrollThreshold;
            } while (scroll_delta <= -guac_mouse.scrollThreshold);

            // Reset delta
            scroll_delta = 0;
        }

        // Down
        if (scroll_delta >= guac_mouse.scrollThreshold) {
            // Repeatedly click the down button until insufficient delta remains
            do {
                guac_mouse.click(Guacamole.Mouse.State.Buttons.DOWN);
                scroll_delta -= guac_mouse.scrollThreshold;
            } while (scroll_delta >= guac_mouse.scrollThreshold);

            // Reset delta
            scroll_delta = 0;
        }

        // All scroll/wheel events must currently be cancelled regardless of
        // whether the dispatched event is cancelled, as there is no Guacamole
        // scroll event and thus no way to cancel scroll events that are
        // smaller than required to produce an up/down click
        Guacamole.Event.DOMEvent.cancelEvent(e);
    }

    element.addEventListener('DOMMouseScroll', mousewheel_handler, false);
    element.addEventListener('mousewheel', mousewheel_handler, false);
    element.addEventListener('wheel', mousewheel_handler, false);

    /**
     * Whether the browser supports CSS3 cursor styling, including hotspot
     * coordinates.
     *
     * @private
     * @type {!boolean}
     */
    var CSS3_CURSOR_SUPPORTED = (function () {
        var div = document.createElement('div');

        // If no cursor property at all, then no support
        if (!('cursor' in div.style)) return false;

        try {
            // Apply simple 1x1 PNG
            div.style.cursor =
                'url(data:image/png;base64,' +
                'iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB' +
                'AQMAAAAl21bKAAAAA1BMVEX///+nxBvI' +
                'AAAACklEQVQI12NgAAAAAgAB4iG8MwAA' +
                'AABJRU5ErkJggg==) 0 0, auto';
        } catch (e) {
            return false;
        }

        // Verify cursor property is set to URL with hotspot
        return /\burl\([^()]*\)\s+0\s+0\b/.test(div.style.cursor || '');
    })();

    /**
     * Changes the local mouse cursor to the given canvas, having the given
     * hotspot coordinates. This affects styling of the element backing this
     * Guacamole.Mouse only, and may fail depending on browser support for
     * setting the mouse cursor.
     *
     * If setting the local cursor is desired, it is up to the implementation
     * to do something else, such as use the software cursor built into
     * Guacamole.Display, if the local cursor cannot be set.
     *
     * @param {!HTMLCanvasElement} canvas
     *     The cursor image.
     *
     * @param {!number} x
     *     The X-coordinate of the cursor hotspot.
     *
     * @param {!number} y
     *     The Y-coordinate of the cursor hotspot.
     *
     * @return {!boolean}
     *     true if the cursor was successfully set, false if the cursor could
     *     not be set for any reason.
     */
    this.setCursor = function (canvas, x, y) {
        // Attempt to set via CSS3 cursor styling
        if (CSS3_CURSOR_SUPPORTED) {
            var dataURL = canvas.toDataURL('image/png');
            element.style.cursor = 'url(' + dataURL + ') ' + x + ' ' + y + ', auto';
            return true;
        }

        // Otherwise, setting cursor failed
        return false;
    };
};

/**
 * The current state of a mouse, including position and buttons.
 *
 * @constructor
 * @augments Guacamole.Position
 * @param {Guacamole.Mouse.State|object} [template={}]
 *     The object whose properties should be copied within the new
 *     Guacamole.Mouse.State.
 */
Guacamole.Mouse.State = function State(template) {
    /**
     * Returns the template object that would be provided to the
     * Guacamole.Mouse.State constructor to produce a new Guacamole.Mouse.State
     * object with the properties specified. The order and type of arguments
     * used by this function are identical to those accepted by the
     * Guacamole.Mouse.State constructor of Apache Guacamole 1.3.0 and older.
     *
     * @private
     * @param {!number} x
     *     The X position of the mouse pointer in pixels.
     *
     * @param {!number} y
     *     The Y position of the mouse pointer in pixels.
     *
     * @param {!boolean} left
     *     Whether the left mouse button is pressed.
     *
     * @param {!boolean} middle
     *     Whether the middle mouse button is pressed.
     *
     * @param {!boolean} right
     *     Whether the right mouse button is pressed.
     *
     * @param {!boolean} up
     *     Whether the up mouse button is pressed (the fourth button, usually
     *     part of a scroll wheel).
     *
     * @param {!boolean} down
     *     Whether the down mouse button is pressed (the fifth button, usually
     *     part of a scroll wheel).
     *
     * @return {!object}
     *     The equivalent template object that would be passed to the new
     *     Guacamole.Mouse.State constructor.
     */
    var legacyConstructor = function legacyConstructor(x, y, left, middle, right, up, down) {
        return {
            x: x,
            y: y,
            left: left,
            middle: middle,
            right: right,
            up: up,
            down: down,
        };
    };

    // Accept old-style constructor, as well
    if (arguments.length > 1) template = legacyConstructor.apply(this, arguments);
    else template = template || {};

    Guacamole.Position.call(this, template);

    /**
     * Whether the left mouse button is currently pressed.
     *
     * @type {!boolean}
     * @default false
     */
    this.left = template.left || false;

    /**
     * Whether the middle mouse button is currently pressed.
     *
     * @type {!boolean}
     * @default false
     */
    this.middle = template.middle || false;

    /**
     * Whether the right mouse button is currently pressed.
     *
     * @type {!boolean}
     * @default false
     */
    this.right = template.right || false;

    /**
     * Whether the up mouse button is currently pressed. This is the fourth
     * mouse button, associated with upward scrolling of the mouse scroll
     * wheel.
     *
     * @type {!boolean}
     * @default false
     */
    this.up = template.up || false;

    /**
     * Whether the down mouse button is currently pressed. This is the fifth
     * mouse button, associated with downward scrolling of the mouse scroll
     * wheel.
     *
     * @type {!boolean}
     * @default false
     */
    this.down = template.down || false;
};

/**
 * All mouse buttons that may be represented by a
 * {@link Guacamole.Mouse.State}.
 *
 * @readonly
 * @enum
 */
Guacamole.Mouse.State.Buttons = {
    /**
     * The name of the {@link Guacamole.Mouse.State} property representing the
     * left mouse button.
     *
     * @constant
     * @type {!string}
     */
    LEFT: 'left',

    /**
     * The name of the {@link Guacamole.Mouse.State} property representing the
     * middle mouse button.
     *
     * @constant
     * @type {!string}
     */
    MIDDLE: 'middle',

    /**
     * The name of the {@link Guacamole.Mouse.State} property representing the
     * right mouse button.
     *
     * @constant
     * @type {!string}
     */
    RIGHT: 'right',

    /**
     * The name of the {@link Guacamole.Mouse.State} property representing the
     * up mouse button (the fourth mouse button, clicked when the mouse scroll
     * wheel is scrolled up).
     *
     * @constant
     * @type {!string}
     */
    UP: 'up',

    /**
     * The name of the {@link Guacamole.Mouse.State} property representing the
     * down mouse button (the fifth mouse button, clicked when the mouse scroll
     * wheel is scrolled up).
     *
     * @constant
     * @type {!string}
     */
    DOWN: 'down',
};

/**
 * Base event type for all mouse events. The mouse producing the event may be
 * the user's local mouse (as with {@link Guacamole.Mouse}) or an emulated
 * mouse (as with {@link Guacamole.Mouse.Touchpad}).
 *
 * @constructor
 * @augments Guacamole.Event.DOMEvent
 * @param {!string} type
 *     The type name of the event ("mousedown", "mouseup", etc.)
 *
 * @param {!Guacamole.Mouse.State} state
 *     The current mouse state.
 *
 * @param {Event|Event[]} [events=[]]
 *     The DOM events that are related to this event, if any.
 */
Guacamole.Mouse.Event = function MouseEvent(type, state, events) {
    Guacamole.Event.DOMEvent.call(this, type, events);

    /**
     * The name of the event handler used by the Guacamole JavaScript API for
     * this event prior to the migration to Guacamole.Event.Target.
     *
     * @private
     * @constant
     * @type {!string}
     */
    var legacyHandlerName = 'on' + this.type;

    /**
     * The current mouse state at the time this event was fired.
     *
     * @type {!Guacamole.Mouse.State}
     */
    this.state = state;

    /**
     * @inheritdoc
     */
    this.invokeLegacyHandler = function invokeLegacyHandler(target) {
        if (target[legacyHandlerName]) {
            this.preventDefault();
            this.stopPropagation();

            target[legacyHandlerName](this.state);
        }
    };
};

/**
 * An object which can dispatch {@link Guacamole.Mouse.Event} objects
 * representing mouse events. These mouse events may be produced from an actual
 * mouse device (as with {@link Guacamole.Mouse}), from an emulated mouse
 * device (as with {@link Guacamole.Mouse.Touchpad}, or may be programmatically
 * generated (using functions like [dispatch()]{@link Guacamole.Mouse.Event.Target#dispatch},
 * [press()]{@link Guacamole.Mouse.Event.Target#press}, and
 * [release()]{@link Guacamole.Mouse.Event.Target#release}).
 *
 * @constructor
 * @augments Guacamole.Event.Target
 */
Guacamole.Mouse.Event.Target = function MouseEventTarget() {
    Guacamole.Event.Target.call(this);

    /**
     * The current mouse state. The properties of this state are updated when
     * mouse events fire. This state object is also passed in as a parameter to
     * the handler of any mouse events.
     *
     * @type {!Guacamole.Mouse.State}
     */
    this.currentState = new Guacamole.Mouse.State();

    /**
     * Fired whenever a mouse button is effectively pressed. Depending on the
     * object dispatching the event, this can be due to a true mouse button
     * press ({@link Guacamole.Mouse}), an emulated mouse button press from a
     * touch gesture ({@link Guacamole.Mouse.Touchpad} and
     * {@link Guacamole.Mouse.Touchscreen}), or may be programmatically
     * generated through [dispatch()]{@link Guacamole.Mouse.Event.Target#dispatch},
     * [press()]{@link Guacamole.Mouse.Event.Target#press}, or
     * [click()]{@link Guacamole.Mouse.Event.Target#click}.
     *
     * @event Guacamole.Mouse.Event.Target#mousedown
     * @param {!Guacamole.Mouse.Event} event
     *     The mousedown event that was fired.
     */

    /**
     * Fired whenever a mouse button is effectively released. Depending on the
     * object dispatching the event, this can be due to a true mouse button
     * release ({@link Guacamole.Mouse}), an emulated mouse button release from
     * a touch gesture ({@link Guacamole.Mouse.Touchpad} and
     * {@link Guacamole.Mouse.Touchscreen}), or may be programmatically
     * generated through [dispatch()]{@link Guacamole.Mouse.Event.Target#dispatch},
     * [release()]{@link Guacamole.Mouse.Event.Target#release}, or
     * [click()]{@link Guacamole.Mouse.Event.Target#click}.
     *
     * @event Guacamole.Mouse.Event.Target#mouseup
     * @param {!Guacamole.Mouse.Event} event
     *     The mouseup event that was fired.
     */

    /**
     * Fired whenever the mouse pointer is effectively moved. Depending on the
     * object dispatching the event, this can be due to true mouse movement
     * ({@link Guacamole.Mouse}), emulated mouse movement from
     * a touch gesture ({@link Guacamole.Mouse.Touchpad} and
     * {@link Guacamole.Mouse.Touchscreen}), or may be programmatically
     * generated through [dispatch()]{@link Guacamole.Mouse.Event.Target#dispatch},
     * or [move()]{@link Guacamole.Mouse.Event.Target#move}.
     *
     * @event Guacamole.Mouse.Event.Target#mousemove
     * @param {!Guacamole.Mouse.Event} event
     *     The mousemove event that was fired.
     */

    /**
     * Fired whenever the mouse pointer leaves the boundaries of the element
     * being monitored for interaction. This will only ever be automatically
     * fired due to movement of an actual mouse device via
     * {@link Guacamole.Mouse} unless programmatically generated through
     * [dispatch()]{@link Guacamole.Mouse.Event.Target#dispatch},
     * or [out()]{@link Guacamole.Mouse.Event.Target#out}.
     *
     * @event Guacamole.Mouse.Event.Target#mouseout
     * @param {!Guacamole.Mouse.Event} event
     *     The mouseout event that was fired.
     */

    /**
     * Presses the given mouse button, if it isn't already pressed. Valid
     * button names are defined by {@link Guacamole.Mouse.State.Buttons} and
     * correspond to the button-related properties of
     * {@link Guacamole.Mouse.State}.
     *
     * @fires Guacamole.Mouse.Event.Target#mousedown
     *
     * @param {!string} button
     *     The name of the mouse button to press, as defined by
     *     {@link Guacamole.Mouse.State.Buttons}.
     *
     * @param {Event|Event[]} [events=[]]
     *     The DOM events that are related to the mouse button press, if any.
     */
    this.press = function press(button, events) {
        if (!this.currentState[button]) {
            this.currentState[button] = true;
            this.dispatch(new Guacamole.Mouse.Event('mousedown', this.currentState, events));
        }
    };

    /**
     * Releases the given mouse button, if it isn't already released. Valid
     * button names are defined by {@link Guacamole.Mouse.State.Buttons} and
     * correspond to the button-related properties of
     * {@link Guacamole.Mouse.State}.
     *
     * @fires Guacamole.Mouse.Event.Target#mouseup
     *
     * @param {!string} button
     *     The name of the mouse button to release, as defined by
     *     {@link Guacamole.Mouse.State.Buttons}.
     *
     * @param {Event|Event[]} [events=[]]
     *     The DOM events related to the mouse button release, if any.
     */
    this.release = function release(button, events) {
        if (this.currentState[button]) {
            this.currentState[button] = false;
            this.dispatch(new Guacamole.Mouse.Event('mouseup', this.currentState, events));
        }
    };

    /**
     * Clicks (presses and releases) the given mouse button. Valid button
     * names are defined by {@link Guacamole.Mouse.State.Buttons} and
     * correspond to the button-related properties of
     * {@link Guacamole.Mouse.State}.
     *
     * @fires Guacamole.Mouse.Event.Target#mousedown
     * @fires Guacamole.Mouse.Event.Target#mouseup
     *
     * @param {!string} button
     *     The name of the mouse button to click, as defined by
     *     {@link Guacamole.Mouse.State.Buttons}.
     *
     * @param {Event|Event[]} [events=[]]
     *     The DOM events related to the click, if any.
     */
    this.click = function click(button, events) {
        this.press(button, events);
        this.release(button, events);
    };

    /**
     * Moves the mouse to the given coordinates.
     *
     * @fires Guacamole.Mouse.Event.Target#mousemove
     *
     * @param {!(Guacamole.Position|object)} position
     *     The new coordinates of the mouse pointer. This object may be a
     *     {@link Guacamole.Position} or any object with "x" and "y"
     *     properties.
     *
     * @param {Event|Event[]} [events=[]]
     *     The DOM events related to the mouse movement, if any.
     */
    this.move = function move(position, events) {
        if (this.currentState.x !== position.x || this.currentState.y !== position.y) {
            this.currentState.x = position.x;
            this.currentState.y = position.y;
            this.dispatch(new Guacamole.Mouse.Event('mousemove', this.currentState, events));
        }
    };

    /**
     * Notifies event listeners that the mouse pointer has left the boundaries
     * of the area being monitored for mouse events.
     *
     * @fires Guacamole.Mouse.Event.Target#mouseout
     *
     * @param {Event|Event[]} [events=[]]
     *     The DOM events related to the mouse leaving the boundaries of the
     *     monitored object, if any.
     */
    this.out = function out(events) {
        this.dispatch(new Guacamole.Mouse.Event('mouseout', this.currentState, events));
    };

    /**
     * Releases all mouse buttons that are currently pressed. If all mouse
     * buttons have already been released, this function has no effect.
     *
     * @fires Guacamole.Mouse.Event.Target#mouseup
     *
     * @param {Event|Event[]} [events=[]]
     *     The DOM event related to all mouse buttons being released, if any.
     */
    this.reset = function reset(events) {
        for (var button in Guacamole.Mouse.State.Buttons) {
            this.release(Guacamole.Mouse.State.Buttons[button], events);
        }
    };
};

/**
 * Provides cross-browser relative touch event translation for a given element.
 *
 * Touch events are translated into mouse events as if the touches occurred
 * on a touchpad (drag to push the mouse pointer, tap to click).
 *
 * @example
 * var touchpad = new Guacamole.Mouse.Touchpad(client.getDisplay().getElement());
 *
 * // Emulate a mouse using touchpad-style gestures, forwarding all mouse
 * // interaction over Guacamole connection
 * touchpad.onEach(['mousedown', 'mousemove', 'mouseup'], function sendMouseEvent(e) {
 *
 *     // Re-show software mouse cursor if possibly hidden by a prior call to
 *     // showCursor(), such as a "mouseout" event handler that hides the
 *     // cursor
 *     client.getDisplay().showCursor(true);
 *
 *     client.sendMouseState(e.state, true);
 *
 * });
 *
 * @constructor
 * @augments Guacamole.Mouse.Event.Target
 * @param {!Element} element
 *     The Element to use to provide touch events.
 */
Guacamole.Mouse.Touchpad = function Touchpad(element) {
    Guacamole.Mouse.Event.Target.call(this);

    /**
     * The "mouseout" event will never be fired by Guacamole.Mouse.Touchpad.
     *
     * @ignore
     * @event Guacamole.Mouse.Touchpad#mouseout
     */

    /**
     * Reference to this Guacamole.Mouse.Touchpad.
     *
     * @private
     * @type {!Guacamole.Mouse.Touchpad}
     */
    var guac_touchpad = this;

    /**
     * The distance a two-finger touch must move per scrollwheel event, in
     * pixels.
     *
     * @type {!number}
     */
    this.scrollThreshold = 20 * (window.devicePixelRatio || 1);

    /**
     * The maximum number of milliseconds to wait for a touch to end for the
     * gesture to be considered a click.
     *
     * @type {!number}
     */
    this.clickTimingThreshold = 250;

    /**
     * The maximum number of pixels to allow a touch to move for the gesture to
     * be considered a click.
     *
     * @type {!number}
     */
    this.clickMoveThreshold = 10 * (window.devicePixelRatio || 1);

    /**
     * The current mouse state. The properties of this state are updated when
     * mouse events fire. This state object is also passed in as a parameter to
     * the handler of any mouse events.
     *
     * @type {!Guacamole.Mouse.State}
     */
    this.currentState = new Guacamole.Mouse.State();

    var touch_count = 0;
    var last_touch_x = 0;
    var last_touch_y = 0;
    var last_touch_time = 0;
    var pixels_moved = 0;

    var touch_buttons = {
        1: 'left',
        2: 'right',
        3: 'middle',
    };

    var gesture_in_progress = false;
    var click_release_timeout = null;

    element.addEventListener(
        'touchend',
        function (e) {
            e.preventDefault();

            // If we're handling a gesture AND this is the last touch
            if (gesture_in_progress && e.touches.length === 0) {
                var time = new Date().getTime();

                // Get corresponding mouse button
                var button = touch_buttons[touch_count];

                // If mouse already down, release anad clear timeout
                if (guac_touchpad.currentState[button]) {
                    // Fire button up event
                    guac_touchpad.release(button, e);

                    // Clear timeout, if set
                    if (click_release_timeout) {
                        window.clearTimeout(click_release_timeout);
                        click_release_timeout = null;
                    }
                }

                // If single tap detected (based on time and distance)
                if (time - last_touch_time <= guac_touchpad.clickTimingThreshold && pixels_moved < guac_touchpad.clickMoveThreshold) {
                    // Fire button down event
                    guac_touchpad.press(button, e);

                    // Delay mouse up - mouse up should be canceled if
                    // touchstart within timeout.
                    click_release_timeout = window.setTimeout(function () {
                        // Fire button up event
                        guac_touchpad.release(button, e);

                        // Gesture now over
                        gesture_in_progress = false;
                    }, guac_touchpad.clickTimingThreshold);
                }

                // If we're not waiting to see if this is a click, stop gesture
                if (!click_release_timeout) gesture_in_progress = false;
            }
        },
        false
    );

    element.addEventListener(
        'touchstart',
        function (e) {
            e.preventDefault();

            // Track number of touches, but no more than three
            touch_count = Math.min(e.touches.length, 3);

            // Clear timeout, if set
            if (click_release_timeout) {
                window.clearTimeout(click_release_timeout);
                click_release_timeout = null;
            }

            // Record initial touch location and time for touch movement
            // and tap gestures
            if (!gesture_in_progress) {
                // Stop mouse events while touching
                gesture_in_progress = true;

                // Record touch location and time
                var starting_touch = e.touches[0];
                last_touch_x = starting_touch.clientX;
                last_touch_y = starting_touch.clientY;
                last_touch_time = new Date().getTime();
                pixels_moved = 0;
            }
        },
        false
    );

    element.addEventListener(
        'touchmove',
        function (e) {
            e.preventDefault();

            // Get change in touch location
            var touch = e.touches[0];
            var delta_x = touch.clientX - last_touch_x;
            var delta_y = touch.clientY - last_touch_y;

            // Track pixels moved
            pixels_moved += Math.abs(delta_x) + Math.abs(delta_y);

            // If only one touch involved, this is mouse move
            if (touch_count === 1) {
                // Calculate average velocity in Manhatten pixels per millisecond
                var velocity = pixels_moved / (new Date().getTime() - last_touch_time);

                // Scale mouse movement relative to velocity
                var scale = 1 + velocity;

                // Update mouse location
                var position = new Guacamole.Position(guac_touchpad.currentState);
                position.x += delta_x * scale;
                position.y += delta_y * scale;

                // Prevent mouse from leaving screen
                position.x = Math.min(Math.max(0, position.x), element.offsetWidth - 1);
                position.y = Math.min(Math.max(0, position.y), element.offsetHeight - 1);

                // Fire movement event, if defined
                guac_touchpad.move(position, e);

                // Update touch location
                last_touch_x = touch.clientX;
                last_touch_y = touch.clientY;
            }

            // Interpret two-finger swipe as scrollwheel
            else if (touch_count === 2) {
                // If change in location passes threshold for scroll
                if (Math.abs(delta_y) >= guac_touchpad.scrollThreshold) {
                    // Decide button based on Y movement direction
                    var button;
                    if (delta_y > 0) button = 'down';
                    else button = 'up';

                    guac_touchpad.click(button, e);

                    // Only update touch location after a scroll has been
                    // detected
                    last_touch_x = touch.clientX;
                    last_touch_y = touch.clientY;
                }
            }
        },
        false
    );
};

/**
 * Provides cross-browser absolute touch event translation for a given element.
 *
 * Touch events are translated into mouse events as if the touches occurred
 * on a touchscreen (tapping anywhere on the screen clicks at that point,
 * long-press to right-click).
 *
 * @example
 * var touchscreen = new Guacamole.Mouse.Touchscreen(client.getDisplay().getElement());
 *
 * // Emulate a mouse using touchscreen-style gestures, forwarding all mouse
 * // interaction over Guacamole connection
 * touchscreen.onEach(['mousedown', 'mousemove', 'mouseup'], function sendMouseEvent(e) {
 *
 *     // Re-show software mouse cursor if possibly hidden by a prior call to
 *     // showCursor(), such as a "mouseout" event handler that hides the
 *     // cursor
 *     client.getDisplay().showCursor(true);
 *
 *     client.sendMouseState(e.state, true);
 *
 * });
 *
 * @constructor
 * @augments Guacamole.Mouse.Event.Target
 * @param {!Element} element
 *     The Element to use to provide touch events.
 */
Guacamole.Mouse.Touchscreen = function Touchscreen(element) {
    Guacamole.Mouse.Event.Target.call(this);

    /**
     * The "mouseout" event will never be fired by Guacamole.Mouse.Touchscreen.
     *
     * @ignore
     * @event Guacamole.Mouse.Touchscreen#mouseout
     */

    /**
     * Reference to this Guacamole.Mouse.Touchscreen.
     *
     * @private
     * @type {!Guacamole.Mouse.Touchscreen}
     */
    var guac_touchscreen = this;

    /**
     * Whether a gesture is known to be in progress. If false, touch events
     * will be ignored.
     *
     * @private
     * @type {!boolean}
     */
    var gesture_in_progress = false;

    /**
     * The start X location of a gesture.
     *
     * @private
     * @type {number}
     */
    var gesture_start_x = null;

    /**
     * The start Y location of a gesture.
     *
     * @private
     * @type {number}
     */
    var gesture_start_y = null;

    /**
     * The timeout associated with the delayed, cancellable click release.
     *
     * @private
     * @type {number}
     */
    var click_release_timeout = null;

    /**
     * The timeout associated with long-press for right click.
     *
     * @private
     * @type {number}
     */
    var long_press_timeout = null;

    /**
     * The distance a two-finger touch must move per scrollwheel event, in
     * pixels.
     *
     * @type {!number}
     */
    this.scrollThreshold = 20 * (window.devicePixelRatio || 1);

    /**
     * The maximum number of milliseconds to wait for a touch to end for the
     * gesture to be considered a click.
     *
     * @type {!number}
     */
    this.clickTimingThreshold = 250;

    /**
     * The maximum number of pixels to allow a touch to move for the gesture to
     * be considered a click.
     *
     * @type {!number}
     */
    this.clickMoveThreshold = 16 * (window.devicePixelRatio || 1);

    /**
     * The amount of time a press must be held for long press to be
     * detected.
     */
    this.longPressThreshold = 500;

    /**
     * Returns whether the given touch event exceeds the movement threshold for
     * clicking, based on where the touch gesture began.
     *
     * @private
     * @param {!TouchEvent} e
     *     The touch event to check.
     *
     * @return {!boolean}
     *     true if the movement threshold is exceeded, false otherwise.
     */
    function finger_moved(e) {
        var touch = e.touches[0] || e.changedTouches[0];
        var delta_x = touch.clientX - gesture_start_x;
        var delta_y = touch.clientY - gesture_start_y;
        return Math.sqrt(delta_x * delta_x + delta_y * delta_y) >= guac_touchscreen.clickMoveThreshold;
    }

    /**
     * Begins a new gesture at the location of the first touch in the given
     * touch event.
     *
     * @private
     * @param {!TouchEvent} e
     *     The touch event beginning this new gesture.
     */
    function begin_gesture(e) {
        var touch = e.touches[0];
        gesture_in_progress = true;
        gesture_start_x = touch.clientX;
        gesture_start_y = touch.clientY;
    }

    /**
     * End the current gesture entirely. Wait for all touches to be done before
     * resuming gesture detection.
     *
     * @private
     */
    function end_gesture() {
        window.clearTimeout(click_release_timeout);
        window.clearTimeout(long_press_timeout);
        gesture_in_progress = false;
    }

    element.addEventListener(
        'touchend',
        function (e) {
            // Do not handle if no gesture
            if (!gesture_in_progress) return;

            // Ignore if more than one touch
            if (e.touches.length !== 0 || e.changedTouches.length !== 1) {
                end_gesture();
                return;
            }

            // Long-press, if any, is over
            window.clearTimeout(long_press_timeout);

            // Always release mouse button if pressed
            guac_touchscreen.release(Guacamole.Mouse.State.Buttons.LEFT, e);

            // If finger hasn't moved enough to cancel the click
            if (!finger_moved(e)) {
                e.preventDefault();

                // If not yet pressed, press and start delay release
                if (!guac_touchscreen.currentState.left) {
                    var touch = e.changedTouches[0];
                    guac_touchscreen.move(Guacamole.Position.fromClientPosition(element, touch.clientX, touch.clientY));
                    guac_touchscreen.press(Guacamole.Mouse.State.Buttons.LEFT, e);

                    // Release button after a delay, if not canceled
                    click_release_timeout = window.setTimeout(function () {
                        guac_touchscreen.release(Guacamole.Mouse.State.Buttons.LEFT, e);
                        end_gesture();
                    }, guac_touchscreen.clickTimingThreshold);
                }
            } // end if finger not moved
        },
        false
    );

    element.addEventListener(
        'touchstart',
        function (e) {
            // Ignore if more than one touch
            if (e.touches.length !== 1) {
                end_gesture();
                return;
            }

            e.preventDefault();

            // New touch begins a new gesture
            begin_gesture(e);

            // Keep button pressed if tap after left click
            window.clearTimeout(click_release_timeout);

            // Click right button if this turns into a long-press
            long_press_timeout = window.setTimeout(function () {
                var touch = e.touches[0];
                guac_touchscreen.move(Guacamole.Position.fromClientPosition(element, touch.clientX, touch.clientY));
                guac_touchscreen.click(Guacamole.Mouse.State.Buttons.RIGHT, e);
                end_gesture();
            }, guac_touchscreen.longPressThreshold);
        },
        false
    );

    element.addEventListener(
        'touchmove',
        function (e) {
            // Do not handle if no gesture
            if (!gesture_in_progress) return;

            // Cancel long press if finger moved
            if (finger_moved(e)) window.clearTimeout(long_press_timeout);

            // Ignore if more than one touch
            if (e.touches.length !== 1) {
                end_gesture();
                return;
            }

            // Update mouse position if dragging
            if (guac_touchscreen.currentState.left) {
                e.preventDefault();

                // Update state
                var touch = e.touches[0];
                guac_touchscreen.move(Guacamole.Position.fromClientPosition(element, touch.clientX, touch.clientY), e);
            }
        },
        false
    );
};
/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

/**
 * An object used by the Guacamole client to house arbitrarily-many named
 * input and output streams.
 *
 * @constructor
 * @param {!Guacamole.Client} client
 *     The client owning this object.
 *
 * @param {!number} index
 *     The index of this object.
 */
Guacamole.Object = function guacamoleObject(client, index) {
    /**
     * Reference to this Guacamole.Object.
     *
     * @private
     * @type {!Guacamole.Object}
     */
    var guacObject = this;

    /**
     * Map of stream name to corresponding queue of callbacks. The queue of
     * callbacks is guaranteed to be in order of request.
     *
     * @private
     * @type {!Object.<string, function[]>}
     */
    var bodyCallbacks = {};

    /**
     * Removes and returns the callback at the head of the callback queue for
     * the stream having the given name. If no such callbacks exist, null is
     * returned.
     *
     * @private
     * @param {!string} name
     *     The name of the stream to retrieve a callback for.
     *
     * @returns {function}
     *     The next callback associated with the stream having the given name,
     *     or null if no such callback exists.
     */
    var dequeueBodyCallback = function dequeueBodyCallback(name) {
        // If no callbacks defined, simply return null
        var callbacks = bodyCallbacks[name];
        if (!callbacks) return null;

        // Otherwise, pull off first callback, deleting the queue if empty
        var callback = callbacks.shift();
        if (callbacks.length === 0) delete bodyCallbacks[name];

        // Return found callback
        return callback;
    };

    /**
     * Adds the given callback to the tail of the callback queue for the stream
     * having the given name.
     *
     * @private
     * @param {!string} name
     *     The name of the stream to associate with the given callback.
     *
     * @param {!function} callback
     *     The callback to add to the queue of the stream with the given name.
     */
    var enqueueBodyCallback = function enqueueBodyCallback(name, callback) {
        // Get callback queue by name, creating first if necessary
        var callbacks = bodyCallbacks[name];
        if (!callbacks) {
            callbacks = [];
            bodyCallbacks[name] = callbacks;
        }

        // Add callback to end of queue
        callbacks.push(callback);
    };

    /**
     * The index of this object.
     *
     * @type {!number}
     */
    this.index = index;

    /**
     * Called when this object receives the body of a requested input stream.
     * By default, all objects will invoke the callbacks provided to their
     * requestInputStream() functions based on the name of the stream
     * requested. This behavior can be overridden by specifying a different
     * handler here.
     *
     * @event
     * @param {!Guacamole.InputStream} inputStream
     *     The input stream of the received body.
     *
     * @param {!string} mimetype
     *     The mimetype of the data being received.
     *
     * @param {!string} name
     *     The name of the stream whose body has been received.
     */
    this.onbody = function defaultBodyHandler(inputStream, mimetype, name) {
        // Call queued callback for the received body, if any
        var callback = dequeueBodyCallback(name);
        if (callback) callback(inputStream, mimetype);
    };

    /**
     * Called when this object is being undefined. Once undefined, no further
     * communication involving this object may occur.
     *
     * @event
     */
    this.onundefine = null;

    /**
     * Requests read access to the input stream having the given name. If
     * successful, a new input stream will be created.
     *
     * @param {!string} name
     *     The name of the input stream to request.
     *
     * @param {function} [bodyCallback]
     *     The callback to invoke when the body of the requested input stream
     *     is received. This callback will be provided a Guacamole.InputStream
     *     and its mimetype as its two only arguments. If the onbody handler of
     *     this object is overridden, this callback will not be invoked.
     */
    this.requestInputStream = function requestInputStream(name, bodyCallback) {
        // Queue body callback if provided
        if (bodyCallback) enqueueBodyCallback(name, bodyCallback);

        // Send request for input stream
        client.requestObjectInputStream(guacObject.index, name);
    };

    /**
     * Creates a new output stream associated with this object and having the
     * given mimetype and name. The legality of a mimetype and name is dictated
     * by the object itself.
     *
     * @param {!string} mimetype
     *     The mimetype of the data which will be sent to the output stream.
     *
     * @param {!string} name
     *     The defined name of an output stream within this object.
     *
     * @returns {!Guacamole.OutputStream}
     *     An output stream which will write blobs to the named output stream
     *     of this object.
     */
    this.createOutputStream = function createOutputStream(mimetype, name) {
        return client.createObjectOutputStream(guacObject.index, mimetype, name);
    };
};

/**
 * The reserved name denoting the root stream of any object. The contents of
 * the root stream MUST be a JSON map of stream name to mimetype.
 *
 * @constant
 * @type {!string}
 */
Guacamole.Object.ROOT_STREAM = '/';

/**
 * The mimetype of a stream containing JSON which maps available stream names
 * to their corresponding mimetype. The root stream of a Guacamole.Object MUST
 * have this mimetype.
 *
 * @constant
 * @type {!string}
 */
Guacamole.Object.STREAM_INDEX_MIMETYPE = 'application/vnd.glyptodon.guacamole.stream-index+json';

/**
 * Dynamic on-screen keyboard. Given the layout object for an on-screen
 * keyboard, this object will construct a clickable on-screen keyboard with its
 * own key events.
 *
 * @constructor
 * @param {!Guacamole.OnScreenKeyboard.Layout} layout
 *     The layout of the on-screen keyboard to display.
 */
Guacamole.OnScreenKeyboard = function (layout) {
    /**
     * Reference to this Guacamole.OnScreenKeyboard.
     *
     * @private
     * @type {!Guacamole.OnScreenKeyboard}
     */
    var osk = this;

    /**
     * Map of currently-set modifiers to the keysym associated with their
     * original press. When the modifier is cleared, this keysym must be
     * released.
     *
     * @private
     * @type {!Object.<String, Number>}
     */
    var modifierKeysyms = {};

    /**
     * Map of all key names to their current pressed states. If a key is not
     * pressed, it may not be in this map at all, but all pressed keys will
     * have a corresponding mapping to true.
     *
     * @private
     * @type {!Object.<String, Boolean>}
     */
    var pressed = {};

    /**
     * All scalable elements which are part of the on-screen keyboard. Each
     * scalable element is carefully controlled to ensure the interface layout
     * and sizing remains constant, even on browsers that would otherwise
     * experience rounding error due to unit conversions.
     *
     * @private
     * @type {!ScaledElement[]}
     */
    var scaledElements = [];

    /**
     * Adds a CSS class to an element.
     *
     * @private
     * @function
     * @param {!Element} element
     *     The element to add a class to.
     *
     * @param {!string} classname
     *     The name of the class to add.
     */
    var addClass = function addClass(element, classname) {
        // If classList supported, use that
        if (element.classList) element.classList.add(classname);
        // Otherwise, simply append the class
        else element.className += ' ' + classname;
    };

    /**
     * Removes a CSS class from an element.
     *
     * @private
     * @function
     * @param {!Element} element
     *     The element to remove a class from.
     *
     * @param {!string} classname
     *     The name of the class to remove.
     */
    var removeClass = function removeClass(element, classname) {
        // If classList supported, use that
        if (element.classList) element.classList.remove(classname);
        // Otherwise, manually filter out classes with given name
        else {
            element.className = element.className.replace(/([^ ]+)[ ]*/g, function removeMatchingClasses(match, testClassname) {
                // If same class, remove
                if (testClassname === classname) return '';

                // Otherwise, allow
                return match;
            });
        }
    };

    /**
     * Counter of mouse events to ignore. This decremented by mousemove, and
     * while non-zero, mouse events will have no effect.
     *
     * @private
     * @type {!number}
     */
    var ignoreMouse = 0;

    /**
     * Ignores all pending mouse events when touch events are the apparent
     * source. Mouse events are ignored until at least touchMouseThreshold
     * mouse events occur without corresponding touch events.
     *
     * @private
     */
    var ignorePendingMouseEvents = function ignorePendingMouseEvents() {
        ignoreMouse = osk.touchMouseThreshold;
    };

    /**
     * An element whose dimensions are maintained according to an arbitrary
     * scale. The conversion factor for these arbitrary units to pixels is
     * provided later via a call to scale().
     *
     * @private
     * @constructor
     * @param {!Element} element
     *     The element whose scale should be maintained.
     *
     * @param {!number} width
     *     The width of the element, in arbitrary units, relative to other
     *     ScaledElements.
     *
     * @param {!number} height
     *     The height of the element, in arbitrary units, relative to other
     *     ScaledElements.
     *
     * @param {boolean} [scaleFont=false]
     *     Whether the line height and font size should be scaled as well.
     */
    var ScaledElement = function ScaledElement(element, width, height, scaleFont) {
        /**
         * The width of this ScaledElement, in arbitrary units, relative to
         * other ScaledElements.
         *
         * @type {!number}
         */
        this.width = width;

        /**
         * The height of this ScaledElement, in arbitrary units, relative to
         * other ScaledElements.
         *
         * @type {!number}
         */
        this.height = height;

        /**
         * Resizes the associated element, updating its dimensions according to
         * the given pixels per unit.
         *
         * @param {!number} pixels
         *     The number of pixels to assign per arbitrary unit.
         */
        this.scale = function (pixels) {
            // Scale element width/height
            element.style.width = width * pixels + 'px';
            element.style.height = height * pixels + 'px';

            // Scale font, if requested
            if (scaleFont) {
                element.style.lineHeight = height * pixels + 'px';
                element.style.fontSize = pixels + 'px';
            }
        };
    };

    /**
     * Returns whether all modifiers having the given names are currently
     * active.
     *
     * @private
     * @param {!string[]} names
     *     The names of all modifiers to test.
     *
     * @returns {!boolean}
     *     true if all specified modifiers are pressed, false otherwise.
     */
    var modifiersPressed = function modifiersPressed(names) {
        // If any required modifiers are not pressed, return false
        for (var i = 0; i < names.length; i++) {
            // Test whether current modifier is pressed
            var name = names[i];
            if (!(name in modifierKeysyms)) return false;
        }

        // Otherwise, all required modifiers are pressed
        return true;
    };

    /**
     * Returns the single matching Key object associated with the key of the
     * given name, where that Key object's requirements (such as pressed
     * modifiers) are all currently satisfied.
     *
     * @private
     * @param {!string} keyName
     *     The name of the key to retrieve.
     *
     * @returns {Guacamole.OnScreenKeyboard.Key}
     *     The Key object associated with the given name, where that object's
     *     requirements are all currently satisfied, or null if no such Key
     *     can be found.
     */
    var getActiveKey = function getActiveKey(keyName) {
        // Get key array for given name
        var keys = osk.keys[keyName];
        if (!keys) return null;

        // Find last matching key
        for (var i = keys.length - 1; i >= 0; i--) {
            // Get candidate key
            var candidate = keys[i];

            // If all required modifiers are pressed, use that key
            if (modifiersPressed(candidate.requires)) return candidate;
        }

        // No valid key
        return null;
    };

    /**
     * Presses the key having the given name, updating the associated key
     * element with the "guac-keyboard-pressed" CSS class. If the key is
     * already pressed, this function has no effect.
     *
     * @private
     * @param {!string} keyName
     *     The name of the key to press.
     *
     * @param {!string} keyElement
     *     The element associated with the given key.
     */
    var press = function press(keyName, keyElement) {
        // Press key if not yet pressed
        if (!pressed[keyName]) {
            addClass(keyElement, 'guac-keyboard-pressed');

            // Get current key based on modifier state
            var key = getActiveKey(keyName);

            // Update modifier state
            if (key.modifier) {
                // Construct classname for modifier
                var modifierClass = 'guac-keyboard-modifier-' + getCSSName(key.modifier);

                // Retrieve originally-pressed keysym, if modifier was already pressed
                var originalKeysym = modifierKeysyms[key.modifier];

                // Activate modifier if not pressed
                if (originalKeysym === undefined) {
                    addClass(keyboard, modifierClass);
                    modifierKeysyms[key.modifier] = key.keysym;

                    // Send key event only if keysym is meaningful
                    if (key.keysym && osk.onkeydown) osk.onkeydown(key.keysym);
                }

                // Deactivate if not pressed
                else {
                    removeClass(keyboard, modifierClass);
                    delete modifierKeysyms[key.modifier];

                    // Send key event only if original keysym is meaningful
                    if (originalKeysym && osk.onkeyup) osk.onkeyup(originalKeysym);
                }
            }

            // If not modifier, send key event now
            else if (osk.onkeydown) osk.onkeydown(key.keysym);

            // Mark key as pressed
            pressed[keyName] = true;
        }
    };

    /**
     * Releases the key having the given name, removing the
     * "guac-keyboard-pressed" CSS class from the associated element. If the
     * key is already released, this function has no effect.
     *
     * @private
     * @param {!string} keyName
     *     The name of the key to release.
     *
     * @param {!string} keyElement
     *     The element associated with the given key.
     */
    var release = function release(keyName, keyElement) {
        // Release key if currently pressed
        if (pressed[keyName]) {
            removeClass(keyElement, 'guac-keyboard-pressed');

            // Get current key based on modifier state
            var key = getActiveKey(keyName);

            // Send key event if not a modifier key
            if (!key.modifier && osk.onkeyup) osk.onkeyup(key.keysym);

            // Mark key as released
            pressed[keyName] = false;
        }
    };

    // Create keyboard
    var keyboard = document.createElement('div');
    keyboard.className = 'guac-keyboard';

    // Do not allow selection or mouse movement to propagate/register.
    keyboard.onselectstart =
        keyboard.onmousemove =
        keyboard.onmouseup =
        keyboard.onmousedown =
            function handleMouseEvents(e) {
                // If ignoring events, decrement counter
                if (ignoreMouse) ignoreMouse--;

                e.stopPropagation();
                return false;
            };

    /**
     * The number of mousemove events to require before re-enabling mouse
     * event handling after receiving a touch event.
     *
     * @type {!number}
     */
    this.touchMouseThreshold = 3;

    /**
     * Fired whenever the user presses a key on this Guacamole.OnScreenKeyboard.
     *
     * @event
     * @param {!number} keysym
     *     The keysym of the key being pressed.
     */
    this.onkeydown = null;

    /**
     * Fired whenever the user releases a key on this Guacamole.OnScreenKeyboard.
     *
     * @event
     * @param {!number} keysym
     *     The keysym of the key being released.
     */
    this.onkeyup = null;

    /**
     * The keyboard layout provided at time of construction.
     *
     * @type {!Guacamole.OnScreenKeyboard.Layout}
     */
    this.layout = new Guacamole.OnScreenKeyboard.Layout(layout);

    /**
     * Returns the element containing the entire on-screen keyboard.
     *
     * @returns {!Element}
     *     The element containing the entire on-screen keyboard.
     */
    this.getElement = function () {
        return keyboard;
    };

    /**
     * Resizes all elements within this Guacamole.OnScreenKeyboard such that
     * the width is close to but does not exceed the specified width. The
     * height of the keyboard is determined based on the width.
     *
     * @param {!number} width
     *     The width to resize this Guacamole.OnScreenKeyboard to, in pixels.
     */
    this.resize = function (width) {
        // Get pixel size of a unit
        var unit = Math.floor((width * 10) / osk.layout.width) / 10;

        // Resize all scaled elements
        for (var i = 0; i < scaledElements.length; i++) {
            var scaledElement = scaledElements[i];
            scaledElement.scale(unit);
        }
    };

    /**
     * Given the name of a key and its corresponding definition, which may be
     * an array of keys objects, a number (keysym), a string (key title), or a
     * single key object, returns an array of key objects, deriving any missing
     * properties as needed, and ensuring the key name is defined.
     *
     * @private
     * @param {!string} name
     *     The name of the key being coerced into an array of Key objects.
     *
     * @param {!(number|string|Guacamole.OnScreenKeyboard.Key|Guacamole.OnScreenKeyboard.Key[])} object
     *     The object defining the behavior of the key having the given name,
     *     which may be the title of the key (a string), the keysym (a number),
     *     a single Key object, or an array of Key objects.
     *
     * @returns {!Guacamole.OnScreenKeyboard.Key[]}
     *     An array of all keys associated with the given name.
     */
    var asKeyArray = function asKeyArray(name, object) {
        // If already an array, just coerce into a true Key[]
        if (object instanceof Array) {
            var keys = [];
            for (var i = 0; i < object.length; i++) {
                keys.push(new Guacamole.OnScreenKeyboard.Key(object[i], name));
            }
            return keys;
        }

        // Derive key object from keysym if that's all we have
        if (typeof object === 'number') {
            return [
                new Guacamole.OnScreenKeyboard.Key({
                    name: name,
                    keysym: object,
                }),
            ];
        }

        // Derive key object from title if that's all we have
        if (typeof object === 'string') {
            return [
                new Guacamole.OnScreenKeyboard.Key({
                    name: name,
                    title: object,
                }),
            ];
        }

        // Otherwise, assume it's already a key object, just not an array
        return [new Guacamole.OnScreenKeyboard.Key(object, name)];
    };

    /**
     * Converts the rather forgiving key mapping allowed by
     * Guacamole.OnScreenKeyboard.Layout into a rigorous mapping of key name
     * to key definition, where the key definition is always an array of Key
     * objects.
     *
     * @private
     * @param {!Object.<string, number|string|Guacamole.OnScreenKeyboard.Key|Guacamole.OnScreenKeyboard.Key[]>} keys
     *     A mapping of key name to key definition, where the key definition is
     *     the title of the key (a string), the keysym (a number), a single
     *     Key object, or an array of Key objects.
     *
     * @returns {!Object.<string, Guacamole.OnScreenKeyboard.Key[]>}
     *     A more-predictable mapping of key name to key definition, where the
     *     key definition is always simply an array of Key objects.
     */
    var getKeys = function getKeys(keys) {
        var keyArrays = {};

        // Coerce all keys into individual key arrays
        for (var name in layout.keys) {
            keyArrays[name] = asKeyArray(name, keys[name]);
        }

        return keyArrays;
    };

    /**
     * Map of all key names to their corresponding set of keys. Each key name
     * may correspond to multiple keys due to the effect of modifiers.
     *
     * @type {!Object.<string, Guacamole.OnScreenKeyboard.Key[]>}
     */
    this.keys = getKeys(layout.keys);

    /**
     * Given an arbitrary string representing the name of some component of the
     * on-screen keyboard, returns a string formatted for use as a CSS class
     * name. The result will be lowercase. Word boundaries previously denoted
     * by CamelCase will be replaced by individual hyphens, as will all
     * contiguous non-alphanumeric characters.
     *
     * @private
     * @param {!string} name
     *     An arbitrary string representing the name of some component of the
     *     on-screen keyboard.
     *
     * @returns {!string}
     *     A string formatted for use as a CSS class name.
     */
    var getCSSName = function getCSSName(name) {
        // Convert name from possibly-CamelCase to hyphenated lowercase
        var cssName = name
            .replace(/([a-z])([A-Z])/g, '$1-$2')
            .replace(/[^A-Za-z0-9]+/g, '-')
            .toLowerCase();

        return cssName;
    };

    /**
     * Appends DOM elements to the given element as dictated by the layout
     * structure object provided. If a name is provided, an additional CSS
     * class, prepended with "guac-keyboard-", will be added to the top-level
     * element.
     *
     * If the layout structure object is an array, all elements within that
     * array will be recursively appended as children of a group, and the
     * top-level element will be given the CSS class "guac-keyboard-group".
     *
     * If the layout structure object is an object, all properties within that
     * object will be recursively appended as children of a group, and the
     * top-level element will be given the CSS class "guac-keyboard-group". The
     * name of each property will be applied as the name of each child object
     * for the sake of CSS. Each property will be added in sorted order.
     *
     * If the layout structure object is a string, the key having that name
     * will be appended. The key will be given the CSS class
     * "guac-keyboard-key" and "guac-keyboard-key-NAME", where NAME is the name
     * of the key. If the name of the key is a single character, this will
     * first be transformed into the C-style hexadecimal literal for the
     * Unicode codepoint of that character. For example, the key "A" would
     * become "guac-keyboard-key-0x41".
     *
     * If the layout structure object is a number, a gap of that size will be
     * inserted. The gap will be given the CSS class "guac-keyboard-gap", and
     * will be scaled according to the same size units as each key.
     *
     * @private
     * @param {!Element} element
     *     The element to append elements to.
     *
     * @param {!(Array|object|string|number)} object
     *     The layout structure object to use when constructing the elements to
     *     append.
     *
     * @param {string} [name]
     *     The name of the top-level element being appended, if any.
     */
    var appendElements = function appendElements(element, object, name) {
        var i;

        // Create div which will become the group or key
        var div = document.createElement('div');

        // Add class based on name, if name given
        if (name) addClass(div, 'guac-keyboard-' + getCSSName(name));

        // If an array, append each element
        if (object instanceof Array) {
            // Add group class
            addClass(div, 'guac-keyboard-group');

            // Append all elements of array
            for (i = 0; i < object.length; i++) appendElements(div, object[i]);
        }

        // If an object, append each property value
        else if (object instanceof Object) {
            // Add group class
            addClass(div, 'guac-keyboard-group');

            // Append all children, sorted by name
            var names = Object.keys(object).sort();
            for (i = 0; i < names.length; i++) {
                var name = names[i];
                appendElements(div, object[name], name);
            }
        }

        // If a number, create as a gap
        else if (typeof object === 'number') {
            // Add gap class
            addClass(div, 'guac-keyboard-gap');

            // Maintain scale
            scaledElements.push(new ScaledElement(div, object, object));
        }

        // If a string, create as a key
        else if (typeof object === 'string') {
            // If key name is only one character, use codepoint for name
            var keyName = object;
            if (keyName.length === 1) keyName = '0x' + keyName.charCodeAt(0).toString(16);

            // Add key container class
            addClass(div, 'guac-keyboard-key-container');

            // Create key element which will contain all possible caps
            var keyElement = document.createElement('div');
            keyElement.className = 'guac-keyboard-key ' + 'guac-keyboard-key-' + getCSSName(keyName);

            // Add all associated keys as caps within DOM
            var keys = osk.keys[object];
            if (keys) {
                for (i = 0; i < keys.length; i++) {
                    // Get current key
                    var key = keys[i];

                    // Create cap element for key
                    var capElement = document.createElement('div');
                    capElement.className = 'guac-keyboard-cap';
                    capElement.textContent = key.title;

                    // Add classes for any requirements
                    for (var j = 0; j < key.requires.length; j++) {
                        var requirement = key.requires[j];
                        addClass(capElement, 'guac-keyboard-requires-' + getCSSName(requirement));
                        addClass(keyElement, 'guac-keyboard-uses-' + getCSSName(requirement));
                    }

                    // Add cap to key within DOM
                    keyElement.appendChild(capElement);
                }
            }

            // Add key to DOM, maintain scale
            div.appendChild(keyElement);
            scaledElements.push(new ScaledElement(div, osk.layout.keyWidths[object] || 1, 1, true));

            /**
             * Handles a touch event which results in the pressing of an OSK
             * key. Touch events will result in mouse events being ignored for
             * touchMouseThreshold events.
             *
             * @private
             * @param {!TouchEvent} e
             *     The touch event being handled.
             */
            var touchPress = function touchPress(e) {
                e.preventDefault();
                ignoreMouse = osk.touchMouseThreshold;
                press(object, keyElement);
            };

            /**
             * Handles a touch event which results in the release of an OSK
             * key. Touch events will result in mouse events being ignored for
             * touchMouseThreshold events.
             *
             * @private
             * @param {!TouchEvent} e
             *     The touch event being handled.
             */
            var touchRelease = function touchRelease(e) {
                e.preventDefault();
                ignoreMouse = osk.touchMouseThreshold;
                release(object, keyElement);
            };

            /**
             * Handles a mouse event which results in the pressing of an OSK
             * key. If mouse events are currently being ignored, this handler
             * does nothing.
             *
             * @private
             * @param {!MouseEvent} e
             *     The touch event being handled.
             */
            var mousePress = function mousePress(e) {
                e.preventDefault();
                if (ignoreMouse === 0) press(object, keyElement);
            };

            /**
             * Handles a mouse event which results in the release of an OSK
             * key. If mouse events are currently being ignored, this handler
             * does nothing.
             *
             * @private
             * @param {!MouseEvent} e
             *     The touch event being handled.
             */
            var mouseRelease = function mouseRelease(e) {
                e.preventDefault();
                if (ignoreMouse === 0) release(object, keyElement);
            };

            // Handle touch events on key
            keyElement.addEventListener('touchstart', touchPress, true);
            keyElement.addEventListener('touchend', touchRelease, true);

            // Handle mouse events on key
            keyElement.addEventListener('mousedown', mousePress, true);
            keyElement.addEventListener('mouseup', mouseRelease, true);
            keyElement.addEventListener('mouseout', mouseRelease, true);
        } // end if object is key name

        // Add newly-created group/key
        element.appendChild(div);
    };

    // Create keyboard layout in DOM
    appendElements(keyboard, layout.layout);
};

/**
 * Represents an entire on-screen keyboard layout, including all available
 * keys, their behaviors, and their relative position and sizing.
 *
 * @constructor
 * @param {!(Guacamole.OnScreenKeyboard.Layout|object)} template
 *     The object whose identically-named properties will be used to initialize
 *     the properties of this layout.
 */
Guacamole.OnScreenKeyboard.Layout = function (template) {
    /**
     * The language of keyboard layout, such as "en_US". This property is for
     * informational purposes only, but it is recommend to conform to the
     * [language code]_[country code] format.
     *
     * @type {!string}
     */
    this.language = template.language;

    /**
     * The type of keyboard layout, such as "qwerty". This property is for
     * informational purposes only, and does not conform to any standard.
     *
     * @type {!string}
     */
    this.type = template.type;

    /**
     * Map of key name to corresponding keysym, title, or key object. If only
     * the keysym or title is provided, the key object will be created
     * implicitly. In all cases, the name property of the key object will be
     * taken from the name given in the mapping.
     *
     * @type {!Object.<string, number|string|Guacamole.OnScreenKeyboard.Key|Guacamole.OnScreenKeyboard.Key[]>}
     */
    this.keys = template.keys;

    /**
     * Arbitrarily nested, arbitrarily grouped key names. The contents of the
     * layout will be traversed to produce an identically-nested grouping of
     * keys in the DOM tree. All strings will be transformed into their
     * corresponding sets of keys, while all objects and arrays will be
     * transformed into named groups and anonymous groups respectively. Any
     * numbers present will be transformed into gaps of that size, scaled
     * according to the same units as each key.
     *
     * @type {!object}
     */
    this.layout = template.layout;

    /**
     * The width of the entire keyboard, in arbitrary units. The width of each
     * key is relative to this width, as both width values are assumed to be in
     * the same units. The conversion factor between these units and pixels is
     * derived later via a call to resize() on the Guacamole.OnScreenKeyboard.
     *
     * @type {!number}
     */
    this.width = template.width;

    /**
     * The width of each key, in arbitrary units, relative to other keys in
     * this layout. The true pixel size of each key will be determined by the
     * overall size of the keyboard. If not defined here, the width of each
     * key will default to 1.
     *
     * @type {!Object.<string, number>}
     */
    this.keyWidths = template.keyWidths || {};
};

/**
 * Represents a single key, or a single possible behavior of a key. Each key
 * on the on-screen keyboard must have at least one associated
 * Guacamole.OnScreenKeyboard.Key, whether that key is explicitly defined or
 * implied, and may have multiple Guacamole.OnScreenKeyboard.Key if behavior
 * depends on modifier states.
 *
 * @constructor
 * @param {!(Guacamole.OnScreenKeyboard.Key|object)} template
 *     The object whose identically-named properties will be used to initialize
 *     the properties of this key.
 *
 * @param {string} [name]
 *     The name to use instead of any name provided within the template, if
 *     any. If omitted, the name within the template will be used, assuming the
 *     template contains a name.
 */
Guacamole.OnScreenKeyboard.Key = function (template, name) {
    /**
     * The unique name identifying this key within the keyboard layout.
     *
     * @type {!string}
     */
    this.name = name || template.name;

    /**
     * The human-readable title that will be displayed to the user within the
     * key. If not provided, this will be derived from the key name.
     *
     * @type {!string}
     */
    this.title = template.title || this.name;

    /**
     * The keysym to be pressed/released when this key is pressed/released. If
     * not provided, this will be derived from the title if the title is a
     * single character.
     *
     * @type {number}
     */
    this.keysym =
        template.keysym ||
        (function deriveKeysym(title) {
            // Do not derive keysym if title is not exactly one character
            if (!title || title.length !== 1) return null;

            // For characters between U+0000 and U+00FF, the keysym is the codepoint
            var charCode = title.charCodeAt(0);
            if (charCode >= 0x0000 && charCode <= 0x00ff) return charCode;

            // For characters between U+0100 and U+10FFFF, the keysym is the codepoint or'd with 0x01000000
            if (charCode >= 0x0100 && charCode <= 0x10ffff) return 0x01000000 | charCode;

            // Unable to derive keysym
            return null;
        })(this.title);

    /**
     * The name of the modifier set when the key is pressed and cleared when
     * this key is released, if any. The names of modifiers are distinct from
     * the names of keys; both the "RightShift" and "LeftShift" keys may set
     * the "shift" modifier, for example. By default, the key will affect no
     * modifiers.
     *
     * @type {string}
     */
    this.modifier = template.modifier;

    /**
     * An array containing the names of each modifier required for this key to
     * have an effect. For example, a lowercase letter may require nothing,
     * while an uppercase letter would require "shift", assuming the Shift key
     * is named "shift" within the layout. By default, the key will require
     * no modifiers.
     *
     * @type {!string[]}
     */
    this.requires = template.requires || [];
};

/**
 * Abstract stream which can receive data.
 *
 * @constructor
 * @param {!Guacamole.Client} client
 *     The client owning this stream.
 *
 * @param {!number} index
 *     The index of this stream.
 */
Guacamole.OutputStream = function (client, index) {
    /**
     * Reference to this stream.
     *
     * @private
     * @type {!Guacamole.OutputStream}
     */
    var guac_stream = this;

    /**
     * The index of this stream.
     * @type {!number}
     */
    this.index = index;

    /**
     * Fired whenever an acknowledgement is received from the server, indicating
     * that a stream operation has completed, or an error has occurred.
     *
     * @event
     * @param {!Guacamole.Status} status
     *     The status of the operation.
     */
    this.onack = null;

    /**
     * Writes the given base64-encoded data to this stream as a blob.
     *
     * @param {!string} data
     *     The base64-encoded data to send.
     */
    this.sendBlob = function (data) {
        client.sendBlob(guac_stream.index, data);
    };

    /**
     * Closes this stream.
     */
    this.sendEnd = function () {
        client.endStream(guac_stream.index);
    };
};

/**
 * Simple Guacamole protocol parser that invokes an oninstruction event when
 * full instructions are available from data received via receive().
 *
 * @constructor
 */
Guacamole.Parser = function () {
    /**
     * Reference to this parser.
     * @private
     */
    var parser = this;

    /**
     * Current buffer of received data. This buffer grows until a full
     * element is available. After a full element is available, that element
     * is flushed into the element buffer.
     *
     * @private
     */
    var buffer = '';

    /**
     * Buffer of all received, complete elements. After an entire instruction
     * is read, this buffer is flushed, and a new instruction begins.
     *
     * @private
     */
    var element_buffer = [];

    // The location of the last element's terminator
    var element_end = -1;

    // Where to start the next length search or the next element
    var start_index = 0;

    /**
     * Appends the given instruction data packet to the internal buffer of
     * this Guacamole.Parser, executing all completed instructions at
     * the beginning of this buffer, if any.
     *
     * @param {!string} packet
     *     The instruction data to receive.
     */
    this.receive = function (packet) {
        // Truncate buffer as necessary
        if (start_index > 4096 && element_end >= start_index) {
            buffer = buffer.substring(start_index);

            // Reset parse relative to truncation
            element_end -= start_index;
            start_index = 0;
        }

        // Append data to buffer
        buffer += packet;

        // While search is within currently received data
        while (element_end < buffer.length) {
            // If we are waiting for element data
            if (element_end >= start_index) {
                // We now have enough data for the element. Parse.
                var element = buffer.substring(start_index, element_end);
                var terminator = buffer.substring(element_end, element_end + 1);

                // Add element to array
                element_buffer.push(element);

                // If last element, handle instruction
                if (terminator == ';') {
                    // Get opcode
                    var opcode = element_buffer.shift();

                    // Call instruction handler.
                    if (parser.oninstruction != null) parser.oninstruction(opcode, element_buffer);

                    // Clear elements
                    element_buffer.length = 0;
                } else if (terminator != ',') throw new Error('Illegal terminator.');

                // Start searching for length at character after
                // element terminator
                start_index = element_end + 1;
            }

            // Search for end of length
            var length_end = buffer.indexOf('.', start_index);
            if (length_end != -1) {
                // Parse length
                var length = parseInt(buffer.substring(element_end + 1, length_end));
                if (isNaN(length)) throw new Error('Non-numeric character in element length.');

                // Calculate start of element
                start_index = length_end + 1;

                // Calculate location of element terminator
                element_end = start_index + length;
            }

            // If no period yet, continue search when more data
            // is received
            else {
                start_index = buffer.length;
                break;
            }
        } // end parse loop
    };

    /**
     * Fired once for every complete Guacamole instruction received, in order.
     *
     * @event
     * @param {!string} opcode
     *     The Guacamole instruction opcode.
     *
     * @param {!string[]} parameters
     *     The parameters provided for the instruction, if any.
     */
    this.oninstruction = null;
};

/**
 * A position in 2-D space.
 *
 * @constructor
 * @param {Guacamole.Position|object} [template={}]
 *     The object whose properties should be copied within the new
 *     Guacamole.Position.
 */
Guacamole.Position = function Position(template) {
    template = template || {};

    /**
     * The current X position, in pixels.
     *
     * @type {!number}
     * @default 0
     */
    this.x = template.x || 0;

    /**
     * The current Y position, in pixels.
     *
     * @type {!number}
     * @default 0
     */
    this.y = template.y || 0;

    /**
     * Assigns the position represented by the given element and
     * clientX/clientY coordinates. The clientX and clientY coordinates are
     * relative to the browser viewport and are commonly available within
     * JavaScript event objects. The final position is translated to
     * coordinates that are relative the given element.
     *
     * @param {!Element} element
     *     The element the coordinates should be relative to.
     *
     * @param {!number} clientX
     *     The viewport-relative X coordinate to translate.
     *
     * @param {!number} clientY
     *     The viewport-relative Y coordinate to translate.
     */
    this.fromClientPosition = function fromClientPosition(element, clientX, clientY) {
        this.x = clientX - element.offsetLeft;
        this.y = clientY - element.offsetTop;

        // This is all JUST so we can get the position within the element
        var parent = element.offsetParent;
        while (parent && !(parent === document.body)) {
            this.x -= parent.offsetLeft - parent.scrollLeft;
            this.y -= parent.offsetTop - parent.scrollTop;

            parent = parent.offsetParent;
        }

        // Element ultimately depends on positioning within document body,
        // take document scroll into account.
        if (parent) {
            var documentScrollLeft = document.body.scrollLeft || document.documentElement.scrollLeft;
            var documentScrollTop = document.body.scrollTop || document.documentElement.scrollTop;

            this.x -= parent.offsetLeft - documentScrollLeft;
            this.y -= parent.offsetTop - documentScrollTop;
        }
    };
};

/**
 * Returns a new {@link Guacamole.Position} representing the relative position
 * of the given clientX/clientY coordinates within the given element. The
 * clientX and clientY coordinates are relative to the browser viewport and are
 * commonly available within JavaScript event objects. The final position is
 * translated to  coordinates that are relative the given element.
 *
 * @param {!Element} element
 *     The element the coordinates should be relative to.
 *
 * @param {!number} clientX
 *     The viewport-relative X coordinate to translate.
 *
 * @param {!number} clientY
 *     The viewport-relative Y coordinate to translate.
 *
 * @returns {!Guacamole.Position}
 *     A new Guacamole.Position representing the relative position of the given
 *     client coordinates.
 */
Guacamole.Position.fromClientPosition = function fromClientPosition(element, clientX, clientY) {
    var position = new Guacamole.Position();
    position.fromClientPosition(element, clientX, clientY);
    return position;
};

/**
 * A description of the format of raw PCM audio, such as that used by
 * Guacamole.RawAudioPlayer and Guacamole.RawAudioRecorder. This object
 * describes the number of bytes per sample, the number of channels, and the
 * overall sample rate.
 *
 * @constructor
 * @param {!(Guacamole.RawAudioFormat|object)} template
 *     The object whose properties should be copied into the corresponding
 *     properties of the new Guacamole.RawAudioFormat.
 */
Guacamole.RawAudioFormat = function RawAudioFormat(template) {
    /**
     * The number of bytes in each sample of audio data. This value is
     * independent of the number of channels.
     *
     * @type {!number}
     */
    this.bytesPerSample = template.bytesPerSample;

    /**
     * The number of audio channels (ie: 1 for mono, 2 for stereo).
     *
     * @type {!number}
     */
    this.channels = template.channels;

    /**
     * The number of samples per second, per channel.
     *
     * @type {!number}
     */
    this.rate = template.rate;
};

/**
 * Parses the given mimetype, returning a new Guacamole.RawAudioFormat
 * which describes the type of raw audio data represented by that mimetype. If
 * the mimetype is not a supported raw audio data mimetype, null is returned.
 *
 * @param {!string} mimetype
 *     The audio mimetype to parse.
 *
 * @returns {Guacamole.RawAudioFormat}
 *     A new Guacamole.RawAudioFormat which describes the type of raw
 *     audio data represented by the given mimetype, or null if the given
 *     mimetype is not supported.
 */
Guacamole.RawAudioFormat.parse = function parseFormat(mimetype) {
    var bytesPerSample;

    // Rate is absolutely required - if null is still present later, the
    // mimetype must not be supported
    var rate = null;

    // Default for both "audio/L8" and "audio/L16" is one channel
    var channels = 1;

    // "audio/L8" has one byte per sample
    if (mimetype.substring(0, 9) === 'audio/L8;') {
        mimetype = mimetype.substring(9);
        bytesPerSample = 1;
    }

    // "audio/L16" has two bytes per sample
    else if (mimetype.substring(0, 10) === 'audio/L16;') {
        mimetype = mimetype.substring(10);
        bytesPerSample = 2;
    }

    // All other types are unsupported
    else return null;

    // Parse all parameters
    var parameters = mimetype.split(',');
    for (var i = 0; i < parameters.length; i++) {
        var parameter = parameters[i];

        // All parameters must have an equals sign separating name from value
        var equals = parameter.indexOf('=');
        if (equals === -1) return null;

        // Parse name and value from parameter string
        var name = parameter.substring(0, equals);
        var value = parameter.substring(equals + 1);

        // Handle each supported parameter
        switch (name) {
            // Number of audio channels
            case 'channels':
                channels = parseInt(value);
                break;

            // Sample rate
            case 'rate':
                rate = parseInt(value);
                break;

            // All other parameters are unsupported
            default:
                return null;
        }
    }

    // The rate parameter is required
    if (rate === null) return null;

    // Return parsed format details
    return new Guacamole.RawAudioFormat({
        bytesPerSample: bytesPerSample,
        channels: channels,
        rate: rate,
    });
};

/**
 * A recording of a Guacamole session. Given a {@link Guacamole.Tunnel} or Blob,
 * the Guacamole.SessionRecording automatically parses Guacamole instructions
 * within the recording source as it plays back the recording. Playback of the
 * recording may be controlled through function calls to the
 * Guacamole.SessionRecording, even while the recording has not yet finished
 * being created or downloaded. Parsing of the contents of the recording will
 * begin immediately and automatically after this constructor is invoked.
 *
 * @constructor
 * @param {!Blob|Guacamole.Tunnel} source
 *     The Blob from which the instructions of the recording should
 *     be read.
 */
Guacamole.SessionRecording = function SessionRecording(source) {
    /**
     * Reference to this Guacamole.SessionRecording.
     *
     * @private
     * @type {!Guacamole.SessionRecording}
     */
    var recording = this;

    /**
     * The Blob from which the instructions of the recording should be read.
     * Note that this value is initialized far below.
     *
     * @private
     * @type {!Blob}
     */
    var recordingBlob;

    /**
     * The tunnel from which the recording should be read, if the recording is
     * being read from a tunnel. If the recording was supplied as a Blob, this
     * will be null.
     *
     * @private
     * @type {Guacamole.Tunnel}
     */
    var tunnel = null;

    /**
     * The number of bytes that this Guacamole.SessionRecording should attempt
     * to read from the given blob in each read operation. Larger blocks will
     * generally read the blob more quickly, but may result in excessive
     * time being spent within the parser, making the page unresponsive
     * while the recording is loading.
     *
     * @private
     * @constant
     * @type {Number}
     */
    var BLOCK_SIZE = 262144;

    /**
     * The minimum number of characters which must have been read between
     * keyframes.
     *
     * @private
     * @constant
     * @type {Number}
     */
    var KEYFRAME_CHAR_INTERVAL = 16384;

    /**
     * The minimum number of milliseconds which must elapse between keyframes.
     *
     * @private
     * @constant
     * @type {Number}
     */
    var KEYFRAME_TIME_INTERVAL = 5000;

    /**
     * All frames parsed from the provided blob.
     *
     * @private
     * @type {!Guacamole.SessionRecording._Frame[]}
     */
    var frames = [];

    /**
     * The timestamp of the last frame which was flagged for use as a keyframe.
     * If no timestamp has yet been flagged, this will be 0.
     *
     * @private
     * @type {!number}
     */
    var lastKeyframe = 0;

    /**
     * Tunnel which feeds arbitrary instructions to the client used by this
     * Guacamole.SessionRecording for playback of the session recording.
     *
     * @private
     * @type {!Guacamole.SessionRecording._PlaybackTunnel}
     */
    var playbackTunnel = new Guacamole.SessionRecording._PlaybackTunnel();

    /**
     * Guacamole.Client instance used for visible playback of the session
     * recording.
     *
     * @private
     * @type {!Guacamole.Client}
     */
    var playbackClient = new Guacamole.Client(playbackTunnel);

    /**
     * The current frame rendered within the playback client. If no frame is
     * yet rendered, this will be -1.
     *
     * @private
     * @type {!number}
     */
    var currentFrame = -1;

    /**
     * The timestamp of the frame when playback began, in milliseconds. If
     * playback is not in progress, this will be null.
     *
     * @private
     * @type {number}
     */
    var startVideoTimestamp = null;

    /**
     * The real-world timestamp when playback began, in milliseconds. If
     * playback is not in progress, this will be null.
     *
     * @private
     * @type {number}
     */
    var startRealTimestamp = null;

    /**
     * An object containing a single "aborted" property which is set to
     * true if the in-progress seek operation should be aborted. If no seek
     * operation is in progress, this will be null.
     *
     * @private
     * @type {object}
     */
    var activeSeek = null;

    /**
     * The byte offset within the recording blob of the first character of
     * the first instruction of the current frame. Here, "current frame"
     * refers to the frame currently being parsed when the provided
     * recording is initially loading. If the recording is not being
     * loaded, this value has no meaning.
     *
     * @private
     * @type {!number}
     */
    var frameStart = 0;

    /**
     * The byte offset within the recording blob of the character which
     * follows the last character of the most recently parsed instruction
     * of the current frame. Here, "current frame" refers to the frame
     * currently being parsed when the provided recording is initially
     * loading. If the recording is not being loaded, this value has no
     * meaning.
     *
     * @private
     * @type {!number}
     */
    var frameEnd = 0;

    /**
     * Whether the initial loading process has been aborted. If the loading
     * process has been aborted, no further blocks of data should be read
     * from the recording.
     *
     * @private
     * @type {!boolean}
     */
    var aborted = false;

    /**
     * The function to invoke when the seek operation initiated by a call
     * to seek() is cancelled or successfully completed. If no seek
     * operation is in progress, this will be null.
     *
     * @private
     * @type {function}
     */
    var seekCallback = null;

    /**
     * Parses all Guacamole instructions within the given blob, invoking
     * the provided instruction callback for each such instruction. Once
     * the end of the blob has been reached (no instructions remain to be
     * parsed), the provided completion callback is invoked. If a parse
     * error prevents reading instructions from the blob, the onerror
     * callback of the Guacamole.SessionRecording is invoked, and no further
     * data is handled within the blob.
     *
     * @private
     * @param {!Blob} blob
     *     The blob to parse Guacamole instructions from.
     *
     * @param {function} [instructionCallback]
     *     The callback to invoke for each Guacamole instruction read from
     *     the given blob. This function must accept the same arguments
     *     as the oninstruction handler of Guacamole.Parser.
     *
     * @param {function} [completionCallback]
     *     The callback to invoke once all instructions have been read from
     *     the given blob.
     */
    var parseBlob = function parseBlob(blob, instructionCallback, completionCallback) {
        // Do not read any further blocks if loading has been aborted
        if (aborted && blob === recordingBlob) return;

        // Prepare a parser to handle all instruction data within the blob,
        // automatically invoking the provided instruction callback for all
        // parsed instructions
        var parser = new Guacamole.Parser();
        parser.oninstruction = instructionCallback;

        var offset = 0;
        var reader = new FileReader();

        /**
         * Reads the block of data at offset bytes within the blob. If no
         * such block exists, then the completion callback provided to
         * parseBlob() is invoked as all data has been read.
         *
         * @private
         */
        var readNextBlock = function readNextBlock() {
            // Do not read any further blocks if loading has been aborted
            if (aborted && blob === recordingBlob) return;

            // Parse all instructions within the block, invoking the
            // onerror handler if a parse error occurs
            if (reader.readyState === 2 /* DONE */) {
                try {
                    parser.receive(reader.result);
                } catch (parseError) {
                    if (recording.onerror) {
                        recording.onerror(parseError.message);
                    }
                    return;
                }
            }

            // If no data remains, the read operation is complete and no
            // further blocks need to be read
            if (offset >= blob.size) {
                if (completionCallback) completionCallback();
            }

            // Otherwise, read the next block
            else {
                var block = blob.slice(offset, offset + BLOCK_SIZE);
                offset += block.size;
                reader.readAsText(block);
            }
        };

        // Read blocks until the end of the given blob is reached
        reader.onload = readNextBlock;
        readNextBlock();
    };

    /**
     * Calculates the size of the given Guacamole instruction element, in
     * Unicode characters. The size returned includes the characters which
     * make up the length, the "." separator between the length and the
     * element itself, and the "," or ";" terminator which follows the
     * element.
     *
     * @private
     * @param {!string} value
     *     The value of the element which has already been parsed (lacks
     *     the initial length, "." separator, and "," or ";" terminator).
     *
     * @returns {!number}
     *     The number of Unicode characters which would make up the given
     *     element within a Guacamole instruction.
     */
    var getElementSize = function getElementSize(value) {
        var valueLength = value.length;

        // Calculate base size, assuming at least one digit, the "."
        // separator, and the "," or ";" terminator
        var protocolSize = valueLength + 3;

        // Add one character for each additional digit that would occur
        // in the element length prefix
        while (valueLength >= 10) {
            protocolSize++;
            valueLength = Math.floor(valueLength / 10);
        }

        return protocolSize;
    };

    // Start playback client connected
    playbackClient.connect();

    // Hide cursor unless mouse position is received
    playbackClient.getDisplay().showCursor(false);

    /**
     * Handles a newly-received instruction, whether from the main Blob or a
     * tunnel, adding new frames and keyframes as necessary. Load progress is
     * reported via onprogress automatically.
     *
     * @private
     * @param {!string} opcode
     *     The opcode of the instruction to handle.
     *
     * @param {!string[]} args
     *     The arguments of the received instruction, if any.
     */
    var loadInstruction = function loadInstruction(opcode, args) {
        // Advance end of frame by overall length of parsed instruction
        frameEnd += getElementSize(opcode);
        for (var i = 0; i < args.length; i++) frameEnd += getElementSize(args[i]);

        // Once a sync is received, store all instructions since the last
        // frame as a new frame
        if (opcode === 'sync') {
            // Parse frame timestamp from sync instruction
            var timestamp = parseInt(args[0]);

            // Add a new frame containing the instructions read since last frame
            var frame = new Guacamole.SessionRecording._Frame(timestamp, frameStart, frameEnd);
            frames.push(frame);
            frameStart = frameEnd;

            // This frame should eventually become a keyframe if enough data
            // has been processed and enough recording time has elapsed, or if
            // this is the absolute first frame
            if (
                frames.length === 1 ||
                (frameEnd - frames[lastKeyframe].start >= KEYFRAME_CHAR_INTERVAL && timestamp - frames[lastKeyframe].timestamp >= KEYFRAME_TIME_INTERVAL)
            ) {
                frame.keyframe = true;
                lastKeyframe = frames.length - 1;
            }

            // Notify that additional content is available
            if (recording.onprogress) recording.onprogress(recording.getDuration(), frameEnd);
        }
    };

    /**
     * Notifies that the session recording has been fully loaded. If the onload
     * handler has not been defined, this function has no effect.
     *
     * @private
     */
    var notifyLoaded = function notifyLoaded() {
        if (recording.onload) recording.onload();
    };

    // Read instructions from provided blob, extracting each frame
    if (source instanceof Blob) parseBlob(recordingBlob, loadInstruction, notifyLoaded);
    // If tunnel provided instead of Blob, extract frames, etc. as instructions
    // are received, buffering things into a Blob for future seeks
    else {
        tunnel = source;
        recordingBlob = new Blob();

        var errorEncountered = false;
        var instructionBuffer = '';

        // Read instructions from provided tunnel, extracting each frame
        tunnel.oninstruction = function handleInstruction(opcode, args) {
            // Reconstitute received instruction
            instructionBuffer += opcode.length + '.' + opcode;
            args.forEach(function appendArg(arg) {
                instructionBuffer += ',' + arg.length + '.' + arg;
            });
            instructionBuffer += ';';

            // Append to Blob (creating a new Blob in the process)
            if (instructionBuffer.length >= BLOCK_SIZE) {
                recordingBlob = new Blob([recordingBlob, instructionBuffer]);
                instructionBuffer = '';
            }

            // Load parsed instruction into recording
            loadInstruction(opcode, args);
        };

        // Report any errors encountered
        tunnel.onerror = function tunnelError(status) {
            errorEncountered = true;
            if (recording.onerror) recording.onerror(status.message);
        };

        tunnel.onstatechange = function tunnelStateChanged(state) {
            if (state === Guacamole.Tunnel.State.CLOSED) {
                // Append any remaining instructions
                if (instructionBuffer.length) {
                    recordingBlob = new Blob([recordingBlob, instructionBuffer]);
                    instructionBuffer = '';
                }

                // Consider recording loaded if tunnel has closed without errors
                if (!errorEncountered) notifyLoaded();
            }
        };
    }

    /**
     * Converts the given absolute timestamp to a timestamp which is relative
     * to the first frame in the recording.
     *
     * @private
     * @param {!number} timestamp
     *     The timestamp to convert to a relative timestamp.
     *
     * @returns {!number}
     *     The difference in milliseconds between the given timestamp and the
     *     first frame of the recording, or zero if no frames yet exist.
     */
    var toRelativeTimestamp = function toRelativeTimestamp(timestamp) {
        // If no frames yet exist, all timestamps are zero
        if (frames.length === 0) return 0;

        // Calculate timestamp relative to first frame
        return timestamp - frames[0].timestamp;
    };

    /**
     * Searches through the given region of frames for the frame having a
     * relative timestamp closest to the timestamp given.
     *
     * @private
     * @param {!number} minIndex
     *     The index of the first frame in the region (the frame having the
     *     smallest timestamp).
     *
     * @param {!number} maxIndex
     *     The index of the last frame in the region (the frame having the
     *     largest timestamp).
     *
     * @param {!number} timestamp
     *     The relative timestamp to search for, where zero denotes the first
     *     frame in the recording.
     *
     * @returns {!number}
     *     The index of the frame having a relative timestamp closest to the
     *     given value.
     */
    var findFrame = function findFrame(minIndex, maxIndex, timestamp) {
        // Do not search if the region contains only one element
        if (minIndex === maxIndex) return minIndex;

        // Split search region into two halves
        var midIndex = Math.floor((minIndex + maxIndex) / 2);
        var midTimestamp = toRelativeTimestamp(frames[midIndex].timestamp);

        // If timestamp is within lesser half, search again within that half
        if (timestamp < midTimestamp && midIndex > minIndex) return findFrame(minIndex, midIndex - 1, timestamp);

        // If timestamp is within greater half, search again within that half
        if (timestamp > midTimestamp && midIndex < maxIndex) return findFrame(midIndex + 1, maxIndex, timestamp);

        // Otherwise, we lucked out and found a frame with exactly the
        // desired timestamp
        return midIndex;
    };

    /**
     * Replays the instructions associated with the given frame, sending those
     * instructions to the playback client.
     *
     * @private
     * @param {!number} index
     *     The index of the frame within the frames array which should be
     *     replayed.
     *
     * @param {function} callback
     *     The callback to invoke once replay of the frame has completed.
     */
    var replayFrame = function replayFrame(index, callback) {
        var frame = frames[index];

        // Replay all instructions within the retrieved frame
        parseBlob(
            recordingBlob.slice(frame.start, frame.end),
            function handleInstruction(opcode, args) {
                playbackTunnel.receiveInstruction(opcode, args);
            },
            function replayCompleted() {
                // Store client state if frame is flagged as a keyframe
                if (frame.keyframe && !frame.clientState) {
                    playbackClient.exportState(function storeClientState(state) {
                        frame.clientState = new Blob([JSON.stringify(state)]);
                    });
                }

                // Update state to correctly represent the current frame
                currentFrame = index;

                if (callback) callback();
            }
        );
    };

    /**
     * Moves the playback position to the given frame, resetting the state of
     * the playback client and replaying frames as necessary. The seek
     * operation will proceed asynchronously. If a seek operation is already in
     * progress, that seek is first aborted. The progress of the seek operation
     * can be observed through the onseek handler and the provided callback.
     *
     * @private
     * @param {!number} index
     *     The index of the frame which should become the new playback
     *     position.
     *
     * @param {function} callback
     *     The callback to invoke once the seek operation has completed.
     *
     * @param {number} [nextRealTimestamp]
     *     The timestamp of the point in time that the given frame should be
     *     displayed, as would be returned by new Date().getTime(). If omitted,
     *     the frame will be displayed as soon as possible.
     */
    var seekToFrame = function seekToFrame(index, callback, nextRealTimestamp) {
        // Abort any in-progress seek
        abortSeek();

        // Note that a new seek operation is in progress
        var thisSeek = (activeSeek = {
            aborted: false,
        });

        var startIndex = index;

        // Replay any applicable incremental frames
        var continueReplay = function continueReplay() {
            // Notify of changes in position
            if (recording.onseek && currentFrame > startIndex) {
                recording.onseek(toRelativeTimestamp(frames[currentFrame].timestamp), currentFrame - startIndex, index - startIndex);
            }

            // Cancel seek if aborted
            if (thisSeek.aborted) return;

            // If frames remain, replay the next frame
            if (currentFrame < index) replayFrame(currentFrame + 1, continueReplay);
            // Otherwise, the seek operation is completed
            else callback();
        };

        // Continue replay after requested delay has elapsed, or
        // immediately if no delay was requested
        var continueAfterRequiredDelay = function continueAfterRequiredDelay() {
            var delay = nextRealTimestamp ? Math.max(nextRealTimestamp - new Date().getTime(), 0) : 0;
            if (delay) window.setTimeout(continueReplay, delay);
            else continueReplay();
        };

        // Back up until startIndex represents current state
        for (; startIndex >= 0; startIndex--) {
            var frame = frames[startIndex];

            // If we've reached the current frame, startIndex represents
            // current state by definition
            if (startIndex === currentFrame) break;

            // If frame has associated absolute state, make that frame the
            // current state
            if (frame.clientState) {
                frame.clientState.text().then(function textReady(text) {
                    playbackClient.importState(JSON.parse(text));
                    currentFrame = startIndex;
                    continueAfterRequiredDelay();
                });
                return;
            }
        }

        continueAfterRequiredDelay();
    };

    /**
     * Aborts the seek operation currently in progress, if any. If no seek
     * operation is in progress, this function has no effect.
     *
     * @private
     */
    var abortSeek = function abortSeek() {
        if (activeSeek) {
            activeSeek.aborted = true;
            activeSeek = null;
        }
    };

    /**
     * Advances playback to the next frame in the frames array and schedules
     * playback of the frame following that frame based on their associated
     * timestamps. If no frames exist after the next frame, playback is paused.
     *
     * @private
     */
    var continuePlayback = function continuePlayback() {
        // If frames remain after advancing, schedule next frame
        if (currentFrame + 1 < frames.length) {
            // Pull the upcoming frame
            var next = frames[currentFrame + 1];

            // Calculate the real timestamp corresponding to when the next
            // frame begins
            var nextRealTimestamp = next.timestamp - startVideoTimestamp + startRealTimestamp;

            // Advance to next frame after enough time has elapsed
            seekToFrame(
                currentFrame + 1,
                function frameDelayElapsed() {
                    continuePlayback();
                },
                nextRealTimestamp
            );
        }

        // Otherwise stop playback
        else recording.pause();
    };

    /**
     * Fired when loading of this recording has completed and all frames
     * are available.
     *
     * @event
     */
    this.onload = null;

    /**
     * Fired when an error occurs which prevents the recording from being
     * played back.
     *
     * @event
     * @param {!string} message
     *     A human-readable message describing the error that occurred.
     */
    this.onerror = null;

    /**
     * Fired when further loading of this recording has been explicitly
     * aborted through a call to abort().
     *
     * @event
     */
    this.onabort = null;

    /**
     * Fired when new frames have become available while the recording is
     * being downloaded.
     *
     * @event
     * @param {!number} duration
     *     The new duration of the recording, in milliseconds.
     *
     * @param {!number} parsedSize
     *     The number of bytes that have been loaded/parsed.
     */
    this.onprogress = null;

    /**
     * Fired whenever playback of the recording has started.
     *
     * @event
     */
    this.onplay = null;

    /**
     * Fired whenever playback of the recording has been paused. This may
     * happen when playback is explicitly paused with a call to pause(), or
     * when playback is implicitly paused due to reaching the end of the
     * recording.
     *
     * @event
     */
    this.onpause = null;

    /**
     * Fired whenever the playback position within the recording changes.
     *
     * @event
     * @param {!number} position
     *     The new position within the recording, in milliseconds.
     *
     * @param {!number} current
     *     The number of frames that have been seeked through. If not
     *     seeking through multiple frames due to a call to seek(), this
     *     will be 1.
     *
     * @param {!number} total
     *     The number of frames that are being seeked through in the
     *     current seek operation. If not seeking through multiple frames
     *     due to a call to seek(), this will be 1.
     */
    this.onseek = null;

    /**
     * Connects the underlying tunnel, beginning download of the Guacamole
     * session. Playback of the Guacamole session cannot occur until at least
     * one frame worth of instructions has been downloaded. If the underlying
     * recording source is a Blob, this function has no effect.
     *
     * @param {string} [data]
     *     The data to send to the tunnel when connecting.
     */
    this.connect = function connect(data) {
        if (tunnel) tunnel.connect(data);
    };

    /**
     * Disconnects the underlying tunnel, stopping further download of the
     * Guacamole session. If the underlying recording source is a Blob, this
     * function has no effect.
     */
    this.disconnect = function disconnect() {
        if (tunnel) tunnel.disconnect();
    };

    /**
     * Aborts the loading process, stopping further processing of the
     * provided data. If the underlying recording source is a Guacamole tunnel,
     * it will be disconnected.
     */
    this.abort = function abort() {
        if (!aborted) {
            aborted = true;
            if (recording.onabort) recording.onabort();

            if (tunnel) tunnel.disconnect();
        }
    };

    /**
     * Returns the underlying display of the Guacamole.Client used by this
     * Guacamole.SessionRecording for playback. The display contains an Element
     * which can be added to the DOM, causing the display (and thus playback of
     * the recording) to become visible.
     *
     * @return {!Guacamole.Display}
     *     The underlying display of the Guacamole.Client used by this
     *     Guacamole.SessionRecording for playback.
     */
    this.getDisplay = function getDisplay() {
        return playbackClient.getDisplay();
    };

    /**
     * Returns whether playback is currently in progress.
     *
     * @returns {!boolean}
     *     true if playback is currently in progress, false otherwise.
     */
    this.isPlaying = function isPlaying() {
        return !!startVideoTimestamp;
    };

    /**
     * Returns the current playback position within the recording, in
     * milliseconds, where zero is the start of the recording.
     *
     * @returns {!number}
     *     The current playback position within the recording, in milliseconds.
     */
    this.getPosition = function getPosition() {
        // Position is simply zero if playback has not started at all
        if (currentFrame === -1) return 0;

        // Return current position as a millisecond timestamp relative to the
        // start of the recording
        return toRelativeTimestamp(frames[currentFrame].timestamp);
    };

    /**
     * Returns the duration of this recording, in milliseconds. If the
     * recording is still being downloaded, this value will gradually increase.
     *
     * @returns {!number}
     *     The duration of this recording, in milliseconds.
     */
    this.getDuration = function getDuration() {
        // If no frames yet exist, duration is zero
        if (frames.length === 0) return 0;

        // Recording duration is simply the timestamp of the last frame
        return toRelativeTimestamp(frames[frames.length - 1].timestamp);
    };

    /**
     * Begins continuous playback of the recording downloaded thus far.
     * Playback of the recording will continue until pause() is invoked or
     * until no further frames exist. Playback is initially paused when a
     * Guacamole.SessionRecording is created, and must be explicitly started
     * through a call to this function. If playback is already in progress,
     * this function has no effect. If a seek operation is in progress,
     * playback resumes at the current position, and the seek is aborted as if
     * completed.
     */
    this.play = function play() {
        // If playback is not already in progress and frames remain,
        // begin playback
        if (!recording.isPlaying() && currentFrame + 1 < frames.length) {
            // Notify that playback is starting
            if (recording.onplay) recording.onplay();

            // Store timestamp of playback start for relative scheduling of
            // future frames
            var next = frames[currentFrame + 1];
            startVideoTimestamp = next.timestamp;
            startRealTimestamp = new Date().getTime();

            // Begin playback of video
            continuePlayback();
        }
    };

    /**
     * Seeks to the given position within the recording. If the recording is
     * currently being played back, playback will continue after the seek is
     * performed. If the recording is currently paused, playback will be
     * paused after the seek is performed. If a seek operation is already in
     * progress, that seek is first aborted. The seek operation will proceed
     * asynchronously.
     *
     * @param {!number} position
     *     The position within the recording to seek to, in milliseconds.
     *
     * @param {function} [callback]
     *     The callback to invoke once the seek operation has completed.
     */
    this.seek = function seek(position, callback) {
        // Do not seek if no frames exist
        if (frames.length === 0) return;

        // Abort active seek operation, if any
        recording.cancel();

        // Pause playback, preserving playback state
        var originallyPlaying = recording.isPlaying();
        recording.pause();

        // Restore playback when seek is completed or cancelled
        seekCallback = function restorePlaybackState() {
            // Seek is no longer in progress
            seekCallback = null;

            // Restore playback state
            if (originallyPlaying) {
                recording.play();
                originallyPlaying = null;
            }

            // Notify that seek has completed
            if (callback) callback();
        };

        // Perform seek
        seekToFrame(findFrame(0, frames.length - 1, position), seekCallback);
    };

    /**
     * Cancels the current seek operation, setting the current frame of the
     * recording to wherever the seek operation was able to reach prior to
     * being cancelled. If a callback was provided to seek(), that callback
     * is invoked. If a seek operation is not currently underway, this
     * function has no effect.
     */
    this.cancel = function cancel() {
        if (seekCallback) {
            abortSeek();
            seekCallback();
        }
    };

    /**
     * Pauses playback of the recording, if playback is currently in progress.
     * If playback is not in progress, this function has no effect. If a seek
     * operation is in progress, the seek is aborted. Playback is initially
     * paused when a Guacamole.SessionRecording is created, and must be
     * explicitly started through a call to play().
     */
    this.pause = function pause() {
        // Abort any in-progress seek / playback
        abortSeek();

        // Stop playback only if playback is in progress
        if (recording.isPlaying()) {
            // Notify that playback is stopping
            if (recording.onpause) recording.onpause();

            // Playback is stopped
            startVideoTimestamp = null;
            startRealTimestamp = null;
        }
    };
};

/**
 * A single frame of Guacamole session data. Each frame is made up of the set
 * of instructions used to generate that frame, and the timestamp as dictated
 * by the "sync" instruction terminating the frame. Optionally, a frame may
 * also be associated with a snapshot of Guacamole client state, such that the
 * frame can be rendered without replaying all previous frames.
 *
 * @private
 * @constructor
 * @param {!number} timestamp
 *     The timestamp of this frame, as dictated by the "sync" instruction which
 *     terminates the frame.
 *
 * @param {!number} start
 *     The byte offset within the blob of the first character of the first
 *     instruction of this frame.
 *
 * @param {!number} end
 *     The byte offset within the blob of character which follows the last
 *     character of the last instruction of this frame.
 */
Guacamole.SessionRecording._Frame = function _Frame(timestamp, start, end) {
    /**
     * Whether this frame should be used as a keyframe if possible. This value
     * is purely advisory. The stored clientState must eventually be manually
     * set for the frame to be used as a keyframe. By default, frames are not
     * keyframes.
     *
     * @type {!boolean}
     * @default false
     */
    this.keyframe = false;

    /**
     * The timestamp of this frame, as dictated by the "sync" instruction which
     * terminates the frame.
     *
     * @type {!number}
     */
    this.timestamp = timestamp;

    /**
     * The byte offset within the blob of the first character of the first
     * instruction of this frame.
     *
     * @type {!number}
     */
    this.start = start;

    /**
     * The byte offset within the blob of character which follows the last
     * character of the last instruction of this frame.
     *
     * @type {!number}
     */
    this.end = end;

    /**
     * A snapshot of client state after this frame was rendered, as returned by
     * a call to exportState(), serialized as JSON, and stored within a Blob.
     * Use of Blobs here is required to ensure the browser can make use of
     * larger disk-backed storage if the size of the recording is large. If no
     * such snapshot has been taken, this will be null.
     *
     * @type {Blob}
     * @default null
     */
    this.clientState = null;
};

/**
 * A read-only Guacamole.Tunnel implementation which streams instructions
 * received through explicit calls to its receiveInstruction() function.
 *
 * @private
 * @constructor
 * @augments {Guacamole.Tunnel}
 */
Guacamole.SessionRecording._PlaybackTunnel = function _PlaybackTunnel() {
    /**
     * Reference to this Guacamole.SessionRecording._PlaybackTunnel.
     *
     * @private
     * @type {!Guacamole.SessionRecording._PlaybackTunnel}
     */
    var tunnel = this;

    this.connect = function connect(data) {
        // Do nothing
    };

    this.sendMessage = function sendMessage(elements) {
        // Do nothing
    };

    this.disconnect = function disconnect() {
        // Do nothing
    };

    /**
     * Invokes this tunnel's oninstruction handler, notifying users of this
     * tunnel (such as a Guacamole.Client instance) that an instruction has
     * been received. If the oninstruction handler has not been set, this
     * function has no effect.
     *
     * @param {!string} opcode
     *     The opcode of the Guacamole instruction.
     *
     * @param {!string[]} args
     *     All arguments associated with this Guacamole instruction.
     */
    this.receiveInstruction = function receiveInstruction(opcode, args) {
        if (tunnel.oninstruction) tunnel.oninstruction(opcode, args);
    };
};

/**
 * A Guacamole status. Each Guacamole status consists of a status code, defined
 * by the protocol, and an optional human-readable message, usually only
 * included for debugging convenience.
 *
 * @constructor
 * @param {!number} code
 *     The Guacamole status code, as defined by Guacamole.Status.Code.
 *
 * @param {string} [message]
 *     An optional human-readable message.
 */
Guacamole.Status = function (code, message) {
    /**
     * Reference to this Guacamole.Status.
     *
     * @private
     * @type {!Guacamole.Status}
     */
    var guac_status = this;

    /**
     * The Guacamole status code.
     *
     * @see Guacamole.Status.Code
     * @type {!number}
     */
    this.code = code;

    /**
     * An arbitrary human-readable message associated with this status, if any.
     * The human-readable message is not required, and is generally provided
     * for debugging purposes only. For user feedback, it is better to translate
     * the Guacamole status code into a message.
     *
     * @type {string}
     */
    this.message = message;

    /**
     * Returns whether this status represents an error.
     *
     * @returns {!boolean}
     *     true if this status represents an error, false otherwise.
     */
    this.isError = function () {
        return guac_status.code < 0 || guac_status.code > 0x00ff;
    };
};

/**
 * Enumeration of all Guacamole status codes.
 */
Guacamole.Status.Code = {
    /**
     * The operation succeeded.
     *
     * @type {!number}
     */
    SUCCESS: 0x0000,

    /**
     * The requested operation is unsupported.
     *
     * @type {!number}
     */
    UNSUPPORTED: 0x0100,

    /**
     * The operation could not be performed due to an internal failure.
     *
     * @type {!number}
     */
    SERVER_ERROR: 0x0200,

    /**
     * The operation could not be performed as the server is busy.
     *
     * @type {!number}
     */
    SERVER_BUSY: 0x0201,

    /**
     * The operation could not be performed because the upstream server is not
     * responding.
     *
     * @type {!number}
     */
    UPSTREAM_TIMEOUT: 0x0202,

    /**
     * The operation was unsuccessful due to an error or otherwise unexpected
     * condition of the upstream server.
     *
     * @type {!number}
     */
    UPSTREAM_ERROR: 0x0203,

    /**
     * The operation could not be performed as the requested resource does not
     * exist.
     *
     * @type {!number}
     */
    RESOURCE_NOT_FOUND: 0x0204,

    /**
     * The operation could not be performed as the requested resource is
     * already in use.
     *
     * @type {!number}
     */
    RESOURCE_CONFLICT: 0x0205,

    /**
     * The operation could not be performed as the requested resource is now
     * closed.
     *
     * @type {!number}
     */
    RESOURCE_CLOSED: 0x0206,

    /**
     * The operation could not be performed because the upstream server does
     * not appear to exist.
     *
     * @type {!number}
     */
    UPSTREAM_NOT_FOUND: 0x0207,

    /**
     * The operation could not be performed because the upstream server is not
     * available to service the request.
     *
     * @type {!number}
     */
    UPSTREAM_UNAVAILABLE: 0x0208,

    /**
     * The session within the upstream server has ended because it conflicted
     * with another session.
     *
     * @type {!number}
     */
    SESSION_CONFLICT: 0x0209,

    /**
     * The session within the upstream server has ended because it appeared to
     * be inactive.
     *
     * @type {!number}
     */
    SESSION_TIMEOUT: 0x020a,

    /**
     * The session within the upstream server has been forcibly terminated.
     *
     * @type {!number}
     */
    SESSION_CLOSED: 0x020b,

    /**
     * The operation could not be performed because bad parameters were given.
     *
     * @type {!number}
     */
    CLIENT_BAD_REQUEST: 0x0300,

    /**
     * Permission was denied to perform the operation, as the user is not yet
     * authorized (not yet logged in, for example).
     *
     * @type {!number}
     */
    CLIENT_UNAUTHORIZED: 0x0301,

    /**
     * Permission was denied to perform the operation, and this permission will
     * not be granted even if the user is authorized.
     *
     * @type {!number}
     */
    CLIENT_FORBIDDEN: 0x0303,

    /**
     * The client took too long to respond.
     *
     * @type {!number}
     */
    CLIENT_TIMEOUT: 0x0308,

    /**
     * The client sent too much data.
     *
     * @type {!number}
     */
    CLIENT_OVERRUN: 0x030d,

    /**
     * The client sent data of an unsupported or unexpected type.
     *
     * @type {!number}
     */
    CLIENT_BAD_TYPE: 0x030f,

    /**
     * The operation failed because the current client is already using too
     * many resources.
     *
     * @type {!number}
     */
    CLIENT_TOO_MANY: 0x031d,
};

/**
 * Returns the Guacamole protocol status code which most closely
 * represents the given HTTP status code.
 *
 * @param {!number} status
 *     The HTTP status code to translate into a Guacamole protocol status
 *     code.
 *
 * @returns {!number}
 *     The Guacamole protocol status code which most closely represents the
 *     given HTTP status code.
 */
Guacamole.Status.Code.fromHTTPCode = function fromHTTPCode(status) {
    // Translate status codes with known equivalents
    switch (status) {
        // HTTP 400 - Bad request
        case 400:
            return Guacamole.Status.Code.CLIENT_BAD_REQUEST;

        // HTTP 403 - Forbidden
        case 403:
            return Guacamole.Status.Code.CLIENT_FORBIDDEN;

        // HTTP 404 - Resource not found
        case 404:
            return Guacamole.Status.Code.RESOURCE_NOT_FOUND;

        // HTTP 429 - Too many requests
        case 429:
            return Guacamole.Status.Code.CLIENT_TOO_MANY;

        // HTTP 503 - Server unavailable
        case 503:
            return Guacamole.Status.Code.SERVER_BUSY;
    }

    // Default all other codes to generic internal error
    return Guacamole.Status.Code.SERVER_ERROR;
};

/**
 * Returns the Guacamole protocol status code which most closely
 * represents the given WebSocket status code.
 *
 * @param {!number} code
 *     The WebSocket status code to translate into a Guacamole protocol
 *     status code.
 *
 * @returns {!number}
 *     The Guacamole protocol status code which most closely represents the
 *     given WebSocket status code.
 */
Guacamole.Status.Code.fromWebSocketCode = function fromWebSocketCode(code) {
    // Translate status codes with known equivalents
    switch (code) {
        // Successful disconnect (no error)
        case 1000: // Normal Closure
            return Guacamole.Status.Code.SUCCESS;

        // Codes which indicate the server is not reachable
        case 1006: // Abnormal Closure (also signalled by JavaScript when the connection cannot be opened in the first place)
        case 1015: // TLS Handshake
            return Guacamole.Status.Code.UPSTREAM_NOT_FOUND;

        // Codes which indicate the server is reachable but busy/unavailable
        case 1001: // Going Away
        case 1012: // Service Restart
        case 1013: // Try Again Later
        case 1014: // Bad Gateway
            return Guacamole.Status.Code.UPSTREAM_UNAVAILABLE;
    }

    // Default all other codes to generic internal error
    return Guacamole.Status.Code.SERVER_ERROR;
};

/**
 * A reader which automatically handles the given input stream, returning
 * strictly text data. Note that this object will overwrite any installed event
 * handlers on the given Guacamole.InputStream.
 *
 * @constructor
 * @param {!Guacamole.InputStream} stream
 *     The stream that data will be read from.
 */
Guacamole.StringReader = function (stream) {
    /**
     * Reference to this Guacamole.InputStream.
     *
     * @private
     * @type {!Guacamole.StringReader}
     */
    var guac_reader = this;

    /**
     * Parser for received UTF-8 data.
     *
     * @type {!Guacamole.UTF8Parser}
     */
    var utf8Parser = new Guacamole.UTF8Parser();

    /**
     * Wrapped Guacamole.ArrayBufferReader.
     *
     * @private
     * @type {!Guacamole.ArrayBufferReader}
     */
    var array_reader = new Guacamole.ArrayBufferReader(stream);

    // Receive blobs as strings
    array_reader.ondata = function (buffer) {
        // Decode UTF-8
        var text = utf8Parser.decode(buffer);

        // Call handler, if present
        if (guac_reader.ontext) guac_reader.ontext(text);
    };

    // Simply call onend when end received
    array_reader.onend = function () {
        if (guac_reader.onend) guac_reader.onend();
    };

    /**
     * Fired once for every blob of text data received.
     *
     * @event
     * @param {!string} text
     *     The data packet received.
     */
    this.ontext = null;

    /**
     * Fired once this stream is finished and no further data will be written.
     * @event
     */
    this.onend = null;
};

/**
 * A writer which automatically writes to the given output stream with text
 * data.
 *
 * @constructor
 * @param {!Guacamole.OutputStream} stream
 *     The stream that data will be written to.
 */
Guacamole.StringWriter = function (stream) {
    /**
     * Reference to this Guacamole.StringWriter.
     *
     * @private
     * @type {!Guacamole.StringWriter}
     */
    var guac_writer = this;

    /**
     * Wrapped Guacamole.ArrayBufferWriter.
     *
     * @private
     * @type {!Guacamole.ArrayBufferWriter}
     */
    var array_writer = new Guacamole.ArrayBufferWriter(stream);

    /**
     * Internal buffer for UTF-8 output.
     *
     * @private
     * @type {!Uint8Array}
     */
    var buffer = new Uint8Array(8192);

    /**
     * The number of bytes currently in the buffer.
     *
     * @private
     * @type {!number}
     */
    var length = 0;

    // Simply call onack for acknowledgements
    array_writer.onack = function (status) {
        if (guac_writer.onack) guac_writer.onack(status);
    };

    /**
     * Expands the size of the underlying buffer by the given number of bytes,
     * updating the length appropriately.
     *
     * @private
     * @param {!number} bytes
     *     The number of bytes to add to the underlying buffer.
     */
    function __expand(bytes) {
        // Resize buffer if more space needed
        if (length + bytes >= buffer.length) {
            var new_buffer = new Uint8Array((length + bytes) * 2);
            new_buffer.set(buffer);
            buffer = new_buffer;
        }

        length += bytes;
    }

    /**
     * Appends a single Unicode character to the current buffer, resizing the
     * buffer if necessary. The character will be encoded as UTF-8.
     *
     * @private
     * @param {!number} codepoint
     *     The codepoint of the Unicode character to append.
     */
    function __append_utf8(codepoint) {
        var mask;
        var bytes;

        // 1 byte
        if (codepoint <= 0x7f) {
            mask = 0x00;
            bytes = 1;
        }

        // 2 byte
        else if (codepoint <= 0x7ff) {
            mask = 0xc0;
            bytes = 2;
        }

        // 3 byte
        else if (codepoint <= 0xffff) {
            mask = 0xe0;
            bytes = 3;
        }

        // 4 byte
        else if (codepoint <= 0x1fffff) {
            mask = 0xf0;
            bytes = 4;
        }

        // If invalid codepoint, append replacement character
        else {
            __append_utf8(0xfffd);
            return;
        }

        // Offset buffer by size
        __expand(bytes);
        var offset = length - 1;

        // Add trailing bytes, if any
        for (var i = 1; i < bytes; i++) {
            buffer[offset--] = 0x80 | (codepoint & 0x3f);
            codepoint >>= 6;
        }

        // Set initial byte
        buffer[offset] = mask | codepoint;
    }

    /**
     * Encodes the given string as UTF-8, returning an ArrayBuffer containing
     * the resulting bytes.
     *
     * @private
     * @param {!string} text
     *     The string to encode as UTF-8.
     *
     * @return {!Uint8Array}
     *     The encoded UTF-8 data.
     */
    function __encode_utf8(text) {
        // Fill buffer with UTF-8
        for (var i = 0; i < text.length; i++) {
            var codepoint = text.charCodeAt(i);
            __append_utf8(codepoint);
        }

        // Flush buffer
        if (length > 0) {
            var out_buffer = buffer.subarray(0, length);
            length = 0;
            return out_buffer;
        }
    }

    /**
     * Sends the given text.
     *
     * @param {!string} text
     *     The text to send.
     */
    this.sendText = function (text) {
        if (text.length) array_writer.sendData(__encode_utf8(text));
    };

    /**
     * Signals that no further text will be sent, effectively closing the
     * stream.
     */
    this.sendEnd = function () {
        array_writer.sendEnd();
    };

    /**
     * Fired for received data, if acknowledged by the server.
     *
     * @event
     * @param {!Guacamole.Status} status
     *     The status of the operation.
     */
    this.onack = null;
};

/**
 * Provides cross-browser multi-touch events for a given element. The events of
 * the given element are automatically populated with handlers that translate
 * touch events into a non-browser-specific event provided by the
 * Guacamole.Touch instance.
 *
 * @constructor
 * @augments Guacamole.Event.Target
 * @param {!Element} element
 *     The Element to use to provide touch events.
 */
Guacamole.Touch = function Touch(element) {
    Guacamole.Event.Target.call(this);

    /**
     * Reference to this Guacamole.Touch.
     *
     * @private
     * @type {!Guacamole.Touch}
     */
    var guacTouch = this;

    /**
     * The default X/Y radius of each touch if the device or browser does not
     * expose the size of the contact area.
     *
     * @private
     * @constant
     * @type {!number}
     */
    var DEFAULT_CONTACT_RADIUS = Math.floor(16 * window.devicePixelRatio);

    /**
     * The set of all active touches, stored by their unique identifiers.
     *
     * @type {!Object.<Number, Guacamole.Touch.State>}
     */
    this.touches = {};

    /**
     * The number of active touches currently stored within
     * {@link Guacamole.Touch#touches touches}.
     */
    this.activeTouches = 0;

    /**
     * Fired whenever a new touch contact is initiated on the element
     * associated with this Guacamole.Touch.
     *
     * @event Guacamole.Touch#touchstart
     * @param {!Guacamole.Touch.Event} event
     *     A {@link Guacamole.Touch.Event} object representing the "touchstart"
     *     event.
     */

    /**
     * Fired whenever an established touch contact moves within the element
     * associated with this Guacamole.Touch.
     *
     * @event Guacamole.Touch#touchmove
     * @param {!Guacamole.Touch.Event} event
     *     A {@link Guacamole.Touch.Event} object representing the "touchmove"
     *     event.
     */

    /**
     * Fired whenever an established touch contact is lifted from the element
     * associated with this Guacamole.Touch.
     *
     * @event Guacamole.Touch#touchend
     * @param {!Guacamole.Touch.Event} event
     *     A {@link Guacamole.Touch.Event} object representing the "touchend"
     *     event.
     */

    element.addEventListener(
        'touchstart',
        function touchstart(e) {
            // Fire "ontouchstart" events for all new touches
            for (var i = 0; i < e.changedTouches.length; i++) {
                var changedTouch = e.changedTouches[i];
                var identifier = changedTouch.identifier;

                // Ignore duplicated touches
                if (guacTouch.touches[identifier]) continue;

                var touch = (guacTouch.touches[identifier] = new Guacamole.Touch.State({
                    id: identifier,
                    radiusX: changedTouch.radiusX || DEFAULT_CONTACT_RADIUS,
                    radiusY: changedTouch.radiusY || DEFAULT_CONTACT_RADIUS,
                    angle: changedTouch.angle || 0.0,
                    force:
                        changedTouch.force ||
                        1.0 /* Within JavaScript changedTouch events, a force of 0.0 indicates the device does not support reporting changedTouch force */,
                }));

                guacTouch.activeTouches++;

                touch.fromClientPosition(element, changedTouch.clientX, changedTouch.clientY);
                guacTouch.dispatch(new Guacamole.Touch.Event('touchmove', e, touch));
            }
        },
        false
    );

    element.addEventListener(
        'touchmove',
        function touchstart(e) {
            // Fire "ontouchmove" events for all updated touches
            for (var i = 0; i < e.changedTouches.length; i++) {
                var changedTouch = e.changedTouches[i];
                var identifier = changedTouch.identifier;

                // Ignore any unrecognized touches
                var touch = guacTouch.touches[identifier];
                if (!touch) continue;

                // Update force only if supported by browser (otherwise, assume
                // force is unchanged)
                if (changedTouch.force) touch.force = changedTouch.force;

                // Update touch area, if supported by browser and device
                touch.angle = changedTouch.angle || 0.0;
                touch.radiusX = changedTouch.radiusX || DEFAULT_CONTACT_RADIUS;
                touch.radiusY = changedTouch.radiusY || DEFAULT_CONTACT_RADIUS;

                // Update with any change in position
                touch.fromClientPosition(element, changedTouch.clientX, changedTouch.clientY);
                guacTouch.dispatch(new Guacamole.Touch.Event('touchmove', e, touch));
            }
        },
        false
    );

    element.addEventListener(
        'touchend',
        function touchstart(e) {
            // Fire "ontouchend" events for all updated touches
            for (var i = 0; i < e.changedTouches.length; i++) {
                var changedTouch = e.changedTouches[i];
                var identifier = changedTouch.identifier;

                // Ignore any unrecognized touches
                var touch = guacTouch.touches[identifier];
                if (!touch) continue;

                // Stop tracking this particular touch
                delete guacTouch.touches[identifier];
                guacTouch.activeTouches--;

                // Touch has ended
                touch.force = 0.0;

                // Update with final position
                touch.fromClientPosition(element, changedTouch.clientX, changedTouch.clientY);
                guacTouch.dispatch(new Guacamole.Touch.Event('touchend', e, touch));
            }
        },
        false
    );
};

/**
 * The current state of a touch contact.
 *
 * @constructor
 * @augments Guacamole.Position
 * @param {Guacamole.Touch.State|object} [template={}]
 *     The object whose properties should be copied within the new
 *     Guacamole.Touch.State.
 */
Guacamole.Touch.State = function State(template) {
    template = template || {};

    Guacamole.Position.call(this, template);

    /**
     * An arbitrary integer ID which uniquely identifies this contact relative
     * to other active contacts.
     *
     * @type {!number}
     * @default 0
     */
    this.id = template.id || 0;

    /**
     * The Y radius of the ellipse covering the general area of the touch
     * contact, in pixels.
     *
     * @type {!number}
     * @default 0
     */
    this.radiusX = template.radiusX || 0;

    /**
     * The X radius of the ellipse covering the general area of the touch
     * contact, in pixels.
     *
     * @type {!number}
     * @default 0
     */
    this.radiusY = template.radiusY || 0;

    /**
     * The rough angle of clockwise rotation of the general area of the touch
     * contact, in degrees.
     *
     * @type {!number}
     * @default 0.0
     */
    this.angle = template.angle || 0.0;

    /**
     * The relative force exerted by the touch contact, where 0 is no force
     * (the touch has been lifted) and 1 is maximum force (the maximum amount
     * of force representable by the device).
     *
     * @type {!number}
     * @default 1.0
     */
    this.force = template.force || 1.0;
};

/**
 * An event which represents a change in state of a single touch contact,
 * including the creation or removal of that contact. If multiple contacts are
 * involved in a touch interaction, each contact will be associated with its
 * own event.
 *
 * @constructor
 * @augments Guacamole.Event.DOMEvent
 * @param {!string} type
 *     The name of the touch event type. Possible values are "touchstart",
 *     "touchmove", and "touchend".
 *
 * @param {!TouchEvent} event
 *     The DOM touch event that produced this Guacamole.Touch.Event.
 *
 * @param {!Guacamole.Touch.State} state
 *     The state of the touch contact associated with this event.
 */
Guacamole.Touch.Event = function TouchEvent(type, event, state) {
    Guacamole.Event.DOMEvent.call(this, type, [event]);

    /**
     * The state of the touch contact associated with this event.
     *
     * @type {!Guacamole.Touch.State}
     */
    this.state = state;
};

/**
 * Core object providing abstract communication for Guacamole. This object
 * is a null implementation whose functions do nothing. Guacamole applications
 * should use {@link Guacamole.HTTPTunnel} instead, or implement their own tunnel based
 * on this one.
 *
 * @constructor
 * @see Guacamole.HTTPTunnel
 */
Guacamole.Tunnel = function () {
    /**
     * Connect to the tunnel with the given optional data. This data is
     * typically used for authentication. The format of data accepted is
     * up to the tunnel implementation.
     *
     * @param {string} [data]
     *     The data to send to the tunnel when connecting.
     */
    this.connect = function (data) {};

    /**
     * Disconnect from the tunnel.
     */
    this.disconnect = function () {};

    /**
     * Send the given message through the tunnel to the service on the other
     * side. All messages are guaranteed to be received in the order sent.
     *
     * @param {...*} elements
     *     The elements of the message to send to the service on the other side
     *     of the tunnel.
     */
    this.sendMessage = function (elements) {};

    /**
     * Changes the stored numeric state of this tunnel, firing the onstatechange
     * event if the new state is different and a handler has been defined.
     *
     * @private
     * @param {!number} state
     *     The new state of this tunnel.
     */
    this.setState = function (state) {
        // Notify only if state changes
        if (state !== this.state) {
            this.state = state;
            if (this.onstatechange) this.onstatechange(state);
        }
    };

    /**
     * Changes the stored UUID that uniquely identifies this tunnel, firing the
     * onuuid event if a handler has been defined.
     *
     * @private
     * @param {string} uuid
     *     The new state of this tunnel.
     */
    this.setUUID = function setUUID(uuid) {
        this.uuid = uuid;
        if (this.onuuid) this.onuuid(uuid);
    };

    /**
     * Returns whether this tunnel is currently connected.
     *
     * @returns {!boolean}
     *     true if this tunnel is currently connected, false otherwise.
     */
    this.isConnected = function isConnected() {
        return this.state === Guacamole.Tunnel.State.OPEN || this.state === Guacamole.Tunnel.State.UNSTABLE;
    };

    /**
     * The current state of this tunnel.
     *
     * @type {!number}
     */
    this.state = Guacamole.Tunnel.State.CLOSED;

    /**
     * The maximum amount of time to wait for data to be received, in
     * milliseconds. If data is not received within this amount of time,
     * the tunnel is closed with an error. The default value is 15000.
     *
     * @type {!number}
     */
    this.receiveTimeout = 15000;

    /**
     * The amount of time to wait for data to be received before considering
     * the connection to be unstable, in milliseconds. If data is not received
     * within this amount of time, the tunnel status is updated to warn that
     * the connection appears unresponsive and may close. The default value is
     * 1500.
     *
     * @type {!number}
     */
    this.unstableThreshold = 1500;

    /**
     * The UUID uniquely identifying this tunnel. If not yet known, this will
     * be null.
     *
     * @type {string}
     */
    this.uuid = null;

    /**
     * Fired when the UUID that uniquely identifies this tunnel is known.
     *
     * @event
     * @param {!string}
     *     The UUID uniquely identifying this tunnel.
     */
    this.onuuid = null;

    /**
     * Fired whenever an error is encountered by the tunnel.
     *
     * @event
     * @param {!Guacamole.Status} status
     *     A status object which describes the error.
     */
    this.onerror = null;

    /**
     * Fired whenever the state of the tunnel changes.
     *
     * @event
     * @param {!number} state
     *     The new state of the client.
     */
    this.onstatechange = null;

    /**
     * Fired once for every complete Guacamole instruction received, in order.
     *
     * @event
     * @param {!string} opcode
     *     The Guacamole instruction opcode.
     *
     * @param {!string[]} parameters
     *     The parameters provided for the instruction, if any.
     */
    this.oninstruction = null;
};

/**
 * The Guacamole protocol instruction opcode reserved for arbitrary internal
 * use by tunnel implementations. The value of this opcode is guaranteed to be
 * the empty string (""). Tunnel implementations may use this opcode for any
 * purpose. It is currently used by the HTTP tunnel to mark the end of the HTTP
 * response, and by the WebSocket tunnel to transmit the tunnel UUID and send
 * connection stability test pings/responses.
 *
 * @constant
 * @type {!string}
 */
Guacamole.Tunnel.INTERNAL_DATA_OPCODE = '';

/**
 * All possible tunnel states.
 *
 * @type {!Object.<string, number>}
 */
Guacamole.Tunnel.State = {
    /**
     * A connection is in pending. It is not yet known whether connection was
     * successful.
     *
     * @type {!number}
     */
    CONNECTING: 0,

    /**
     * Connection was successful, and data is being received.
     *
     * @type {!number}
     */
    OPEN: 1,

    /**
     * The connection is closed. Connection may not have been successful, the
     * tunnel may have been explicitly closed by either side, or an error may
     * have occurred.
     *
     * @type {!number}
     */
    CLOSED: 2,

    /**
     * The connection is open, but communication through the tunnel appears to
     * be disrupted, and the connection may close as a result.
     *
     * @type {!number}
     */
    UNSTABLE: 3,
};

/**
 * Guacamole Tunnel implemented over HTTP via XMLHttpRequest.
 *
 * @constructor
 * @augments Guacamole.Tunnel
 *
 * @param {!string} tunnelURL
 *     The URL of the HTTP tunneling service.
 *
 * @param {boolean} [crossDomain=false]
 *     Whether tunnel requests will be cross-domain, and thus must use CORS
 *     mechanisms and headers. By default, it is assumed that tunnel requests
 *     will be made to the same domain.
 *
 * @param {object} [extraTunnelHeaders={}]
 *     Key value pairs containing the header names and values of any additional
 *     headers to be sent in tunnel requests. By default, no extra headers will
 *     be added.
 */
Guacamole.HTTPTunnel = function (tunnelURL, crossDomain, extraTunnelHeaders) {
    /**
     * Reference to this HTTP tunnel.
     *
     * @private
     * @type {!Guacamole.HTTPTunnel}
     */
    var tunnel = this;

    var TUNNEL_CONNECT = tunnelURL + '?connect';
    var TUNNEL_READ = tunnelURL + '?read:';
    var TUNNEL_WRITE = tunnelURL + '?write:';

    var POLLING_ENABLED = 1;
    var POLLING_DISABLED = 0;

    // Default to polling - will be turned off automatically if not needed
    var pollingMode = POLLING_ENABLED;

    var sendingMessages = false;
    var outputMessageBuffer = '';

    // If requests are expected to be cross-domain, the cookie that the HTTP
    // tunnel depends on will only be sent if withCredentials is true
    var withCredentials = !!crossDomain;

    /**
     * The current receive timeout ID, if any.
     *
     * @private
     * @type {number}
     */
    var receive_timeout = null;

    /**
     * The current connection stability timeout ID, if any.
     *
     * @private
     * @type {number}
     */
    var unstableTimeout = null;

    /**
     * The current connection stability test ping interval ID, if any. This
     * will only be set upon successful connection.
     *
     * @private
     * @type {number}
     */
    var pingInterval = null;

    /**
     * The number of milliseconds to wait between connection stability test
     * pings.
     *
     * @private
     * @constant
     * @type {!number}
     */
    var PING_FREQUENCY = 500;

    /**
     * Additional headers to be sent in tunnel requests. This dictionary can be
     * populated with key/value header pairs to pass information such as authentication
     * tokens, etc.
     *
     * @private
     * @type {!object}
     */
    var extraHeaders = extraTunnelHeaders || {};

    /**
     * The name of the HTTP header containing the session token specific to the
     * HTTP tunnel implementation.
     *
     * @private
     * @constant
     * @type {!string}
     */
    var TUNNEL_TOKEN_HEADER = 'Guacamole-Tunnel-Token';

    /**
     * The session token currently assigned to this HTTP tunnel. All distinct
     * HTTP tunnel connections will have their own dedicated session token.
     *
     * @private
     * @type {string}
     */
    var tunnelSessionToken = null;

    /**
     * Adds the configured additional headers to the given request.
     *
     * @private
     * @param {!XMLHttpRequest} request
     *     The request where the configured extra headers will be added.
     *
     * @param {!object} headers
     *     The headers to be added to the request.
     */
    function addExtraHeaders(request, headers) {
        for (var name in headers) {
            request.setRequestHeader(name, headers[name]);
        }
    }

    /**
     * Resets the state of timers tracking network activity and stability. If
     * those timers are not yet started, invoking this function starts them.
     * This function should be invoked when the tunnel is established and every
     * time there is network activity on the tunnel, such that the timers can
     * safely assume the network and/or server are not responding if this
     * function has not been invoked for a significant period of time.
     *
     * @private
     */
    var resetTimers = function resetTimers() {
        // Get rid of old timeouts (if any)
        window.clearTimeout(receive_timeout);
        window.clearTimeout(unstableTimeout);

        // Clear unstable status
        if (tunnel.state === Guacamole.Tunnel.State.UNSTABLE) tunnel.setState(Guacamole.Tunnel.State.OPEN);

        // Set new timeout for tracking overall connection timeout
        receive_timeout = window.setTimeout(function () {
            close_tunnel(new Guacamole.Status(Guacamole.Status.Code.UPSTREAM_TIMEOUT, 'Server timeout.'));
        }, tunnel.receiveTimeout);

        // Set new timeout for tracking suspected connection instability
        unstableTimeout = window.setTimeout(function () {
            tunnel.setState(Guacamole.Tunnel.State.UNSTABLE);
        }, tunnel.unstableThreshold);
    };

    /**
     * Closes this tunnel, signaling the given status and corresponding
     * message, which will be sent to the onerror handler if the status is
     * an error status.
     *
     * @private
     * @param {!Guacamole.Status} status
     *     The status causing the connection to close;
     */
    function close_tunnel(status) {
        // Get rid of old timeouts (if any)
        window.clearTimeout(receive_timeout);
        window.clearTimeout(unstableTimeout);

        // Cease connection test pings
        window.clearInterval(pingInterval);

        // Ignore if already closed
        if (tunnel.state === Guacamole.Tunnel.State.CLOSED) return;

        // If connection closed abnormally, signal error.
        if (status.code !== Guacamole.Status.Code.SUCCESS && tunnel.onerror) {
            // Ignore RESOURCE_NOT_FOUND if we've already connected, as that
            // only signals end-of-stream for the HTTP tunnel.
            if (tunnel.state === Guacamole.Tunnel.State.CONNECTING || status.code !== Guacamole.Status.Code.RESOURCE_NOT_FOUND) tunnel.onerror(status);
        }

        // Reset output message buffer
        sendingMessages = false;

        // Mark as closed
        tunnel.setState(Guacamole.Tunnel.State.CLOSED);
    }

    this.sendMessage = function () {
        // Do not attempt to send messages if not connected
        if (!tunnel.isConnected()) return;

        // Do not attempt to send empty messages
        if (arguments.length === 0) return;

        /**
         * Converts the given value to a length/string pair for use as an
         * element in a Guacamole instruction.
         *
         * @private
         * @param value
         *     The value to convert.
         *
         * @return {!string}
         *     The converted value.
         */
        function getElement(value) {
            var string = new String(value);
            return string.length + '.' + string;
        }

        // Initialized message with first element
        var message = getElement(arguments[0]);

        // Append remaining elements
        for (var i = 1; i < arguments.length; i++) message += ',' + getElement(arguments[i]);

        // Final terminator
        message += ';';

        // Add message to buffer
        outputMessageBuffer += message;

        // Send if not currently sending
        if (!sendingMessages) sendPendingMessages();
    };

    function sendPendingMessages() {
        // Do not attempt to send messages if not connected
        if (!tunnel.isConnected()) return;

        if (outputMessageBuffer.length > 0) {
            sendingMessages = true;

            var message_xmlhttprequest = new XMLHttpRequest();
            message_xmlhttprequest.open('POST', TUNNEL_WRITE + tunnel.uuid);
            message_xmlhttprequest.withCredentials = withCredentials;
            addExtraHeaders(message_xmlhttprequest, extraHeaders);
            message_xmlhttprequest.setRequestHeader('Content-type', 'application/octet-stream');
            message_xmlhttprequest.setRequestHeader(TUNNEL_TOKEN_HEADER, tunnelSessionToken);

            // Once response received, send next queued event.
            message_xmlhttprequest.onreadystatechange = function () {
                if (message_xmlhttprequest.readyState === 4) {
                    resetTimers();

                    // If an error occurs during send, handle it
                    if (message_xmlhttprequest.status !== 200) handleHTTPTunnelError(message_xmlhttprequest);
                    // Otherwise, continue the send loop
                    else sendPendingMessages();
                }
            };

            message_xmlhttprequest.send(outputMessageBuffer);
            outputMessageBuffer = ''; // Clear buffer
        } else sendingMessages = false;
    }

    function handleHTTPTunnelError(xmlhttprequest) {
        // Pull status code directly from headers provided by Guacamole
        var code = parseInt(xmlhttprequest.getResponseHeader('Guacamole-Status-Code'));
        if (code) {
            var message = xmlhttprequest.getResponseHeader('Guacamole-Error-Message');
            close_tunnel(new Guacamole.Status(code, message));
        }

        // Failing that, derive a Guacamole status code from the HTTP status
        // code provided by the browser
        else if (xmlhttprequest.status)
            close_tunnel(new Guacamole.Status(Guacamole.Status.Code.fromHTTPCode(xmlhttprequest.status), xmlhttprequest.statusText));
        // Otherwise, assume server is unreachable
        else close_tunnel(new Guacamole.Status(Guacamole.Status.Code.UPSTREAM_NOT_FOUND));
    }

    function handleResponse(xmlhttprequest) {
        var interval = null;
        var nextRequest = null;

        var dataUpdateEvents = 0;

        // The location of the last element's terminator
        var elementEnd = -1;

        // Where to start the next length search or the next element
        var startIndex = 0;

        // Parsed elements
        var elements = new Array();

        function parseResponse() {
            // Do not handle responses if not connected
            if (!tunnel.isConnected()) {
                // Clean up interval if polling
                if (interval !== null) clearInterval(interval);

                return;
            }

            // Do not parse response yet if not ready
            if (xmlhttprequest.readyState < 2) return;

            // Attempt to read status
            var status;
            try {
                status = xmlhttprequest.status;
            } catch (e) {
                // If status could not be read, assume successful.
                status = 200;
            }

            // Start next request as soon as possible IF request was successful
            if (!nextRequest && status === 200) nextRequest = makeRequest();

            // Parse stream when data is received and when complete.
            if (xmlhttprequest.readyState === 3 || xmlhttprequest.readyState === 4) {
                resetTimers();

                // Also poll every 30ms (some browsers don't repeatedly call onreadystatechange for new data)
                if (pollingMode === POLLING_ENABLED) {
                    if (xmlhttprequest.readyState === 3 && !interval) interval = setInterval(parseResponse, 30);
                    else if (xmlhttprequest.readyState === 4 && interval) clearInterval(interval);
                }

                // If canceled, stop transfer
                if (xmlhttprequest.status === 0) {
                    tunnel.disconnect();
                    return;
                }

                // Halt on error during request
                else if (xmlhttprequest.status !== 200) {
                    handleHTTPTunnelError(xmlhttprequest);
                    return;
                }

                // Attempt to read in-progress data
                var current;
                try {
                    current = xmlhttprequest.responseText;
                } catch (e) {
                    // Do not attempt to parse if data could not be read
                    return;
                }

                // While search is within currently received data
                while (elementEnd < current.length) {
                    // If we are waiting for element data
                    if (elementEnd >= startIndex) {
                        // We now have enough data for the element. Parse.
                        var element = current.substring(startIndex, elementEnd);
                        var terminator = current.substring(elementEnd, elementEnd + 1);

                        // Add element to array
                        elements.push(element);

                        // If last element, handle instruction
                        if (terminator === ';') {
                            // Get opcode
                            var opcode = elements.shift();

                            // Call instruction handler.
                            if (tunnel.oninstruction) tunnel.oninstruction(opcode, elements);

                            // Clear elements
                            elements.length = 0;
                        }

                        // Start searching for length at character after
                        // element terminator
                        startIndex = elementEnd + 1;
                    }

                    // Search for end of length
                    var lengthEnd = current.indexOf('.', startIndex);
                    if (lengthEnd !== -1) {
                        // Parse length
                        var length = parseInt(current.substring(elementEnd + 1, lengthEnd));

                        // If we're done parsing, handle the next response.
                        if (length === 0) {
                            // Clean up interval if polling
                            if (interval) clearInterval(interval);

                            // Clean up object
                            xmlhttprequest.onreadystatechange = null;
                            xmlhttprequest.abort();

                            // Start handling next request
                            if (nextRequest) handleResponse(nextRequest);

                            // Done parsing
                            break;
                        }

                        // Calculate start of element
                        startIndex = lengthEnd + 1;

                        // Calculate location of element terminator
                        elementEnd = startIndex + length;
                    }

                    // If no period yet, continue search when more data
                    // is received
                    else {
                        startIndex = current.length;
                        break;
                    }
                } // end parse loop
            }
        }

        // If response polling enabled, attempt to detect if still
        // necessary (via wrapping parseResponse())
        if (pollingMode === POLLING_ENABLED) {
            xmlhttprequest.onreadystatechange = function () {
                // If we receive two or more readyState==3 events,
                // there is no need to poll.
                if (xmlhttprequest.readyState === 3) {
                    dataUpdateEvents++;
                    if (dataUpdateEvents >= 2) {
                        pollingMode = POLLING_DISABLED;
                        xmlhttprequest.onreadystatechange = parseResponse;
                    }
                }

                parseResponse();
            };
        }

        // Otherwise, just parse
        else xmlhttprequest.onreadystatechange = parseResponse;

        parseResponse();
    }

    /**
     * Arbitrary integer, unique for each tunnel read request.
     * @private
     */
    var request_id = 0;

    function makeRequest() {
        // Make request, increment request ID
        var xmlhttprequest = new XMLHttpRequest();
        xmlhttprequest.open('GET', TUNNEL_READ + tunnel.uuid + ':' + request_id++);
        xmlhttprequest.setRequestHeader(TUNNEL_TOKEN_HEADER, tunnelSessionToken);
        xmlhttprequest.withCredentials = withCredentials;
        addExtraHeaders(xmlhttprequest, extraHeaders);
        xmlhttprequest.send(null);

        return xmlhttprequest;
    }

    this.connect = function (data) {
        // Start waiting for connect
        resetTimers();

        // Mark the tunnel as connecting
        tunnel.setState(Guacamole.Tunnel.State.CONNECTING);

        // Start tunnel and connect
        var connect_xmlhttprequest = new XMLHttpRequest();
        connect_xmlhttprequest.onreadystatechange = function () {
            if (connect_xmlhttprequest.readyState !== 4) return;

            // If failure, throw error
            if (connect_xmlhttprequest.status !== 200) {
                handleHTTPTunnelError(connect_xmlhttprequest);
                return;
            }

            resetTimers();

            // Get UUID and HTTP-specific tunnel session token from response
            tunnel.setUUID(connect_xmlhttprequest.responseText);
            tunnelSessionToken = connect_xmlhttprequest.getResponseHeader(TUNNEL_TOKEN_HEADER);

            // Fail connect attempt if token is not successfully assigned
            if (!tunnelSessionToken) {
                close_tunnel(new Guacamole.Status(Guacamole.Status.Code.UPSTREAM_NOT_FOUND));
                return;
            }

            // Mark as open
            tunnel.setState(Guacamole.Tunnel.State.OPEN);

            // Ping tunnel endpoint regularly to test connection stability
            pingInterval = setInterval(function sendPing() {
                tunnel.sendMessage('nop');
            }, PING_FREQUENCY);

            // Start reading data
            handleResponse(makeRequest());
        };

        connect_xmlhttprequest.open('POST', TUNNEL_CONNECT, true);
        connect_xmlhttprequest.withCredentials = withCredentials;
        addExtraHeaders(connect_xmlhttprequest, extraHeaders);
        connect_xmlhttprequest.setRequestHeader('Content-type', 'application/x-www-form-urlencoded; charset=UTF-8');
        connect_xmlhttprequest.send(data);
    };

    this.disconnect = function () {
        close_tunnel(new Guacamole.Status(Guacamole.Status.Code.SUCCESS, 'Manually closed.'));
    };
};

Guacamole.HTTPTunnel.prototype = new Guacamole.Tunnel();

/**
 * Guacamole Tunnel implemented over WebSocket via XMLHttpRequest.
 *
 * @constructor
 * @augments Guacamole.Tunnel
 * @param {!string} tunnelURL
 *     The URL of the WebSocket tunneling service.
 */
Guacamole.WebSocketTunnel = function (tunnelURL) {
    /**
     * Reference to this WebSocket tunnel.
     *
     * @private
     * @type {Guacamole.WebSocketTunnel}
     */
    var tunnel = this;

    /**
     * The WebSocket used by this tunnel.
     *
     * @private
     * @type {WebSocket}
     */
    var socket = null;

    /**
     * The current receive timeout ID, if any.
     *
     * @private
     * @type {number}
     */
    var receive_timeout = null;

    /**
     * The current connection stability timeout ID, if any.
     *
     * @private
     * @type {number}
     */
    var unstableTimeout = null;

    /**
     * The current connection stability test ping timeout ID, if any. This
     * will only be set upon successful connection.
     *
     * @private
     * @type {number}
     */
    var pingTimeout = null;

    /**
     * The WebSocket protocol corresponding to the protocol used for the current
     * location.
     *
     * @private
     * @type {!Object.<string, string>}
     */
    var ws_protocol = {
        'http:': 'ws:',
        'https:': 'wss:',
    };

    /**
     * The number of milliseconds to wait between connection stability test
     * pings.
     *
     * @private
     * @constant
     * @type {!number}
     */
    var PING_FREQUENCY = 500;

    /**
     * The timestamp of the point in time that the last connection stability
     * test ping was sent, in milliseconds elapsed since midnight of January 1,
     * 1970 UTC.
     *
     * @private
     * @type {!number}
     */
    var lastSentPing = 0;

    // Transform current URL to WebSocket URL

    // If not already a websocket URL
    if (tunnelURL.substring(0, 3) !== 'ws:' && tunnelURL.substring(0, 4) !== 'wss:') {
        var protocol = ws_protocol[window.location.protocol];

        // If absolute URL, convert to absolute WS URL
        if (tunnelURL.substring(0, 1) === '/') tunnelURL = protocol + '//' + window.location.host + tunnelURL;
        // Otherwise, construct absolute from relative URL
        else {
            // Get path from pathname
            var slash = window.location.pathname.lastIndexOf('/');
            var path = window.location.pathname.substring(0, slash + 1);

            // Construct absolute URL
            tunnelURL = protocol + '//' + window.location.host + path + tunnelURL;
        }
    }

    /**
     * Sends an internal "ping" instruction to the Guacamole WebSocket
     * endpoint, verifying network connection stability. If the network is
     * stable, the Guacamole server will receive this instruction and respond
     * with an identical ping.
     *
     * @private
     */
    var sendPing = function sendPing() {
        var currentTime = new Date().getTime();
        tunnel.sendMessage(Guacamole.Tunnel.INTERNAL_DATA_OPCODE, 'ping', currentTime);
        lastSentPing = currentTime;
    };

    /**
     * Resets the state of timers tracking network activity and stability. If
     * those timers are not yet started, invoking this function starts them.
     * This function should be invoked when the tunnel is established and every
     * time there is network activity on the tunnel, such that the timers can
     * safely assume the network and/or server are not responding if this
     * function has not been invoked for a significant period of time.
     *
     * @private
     */
    var resetTimers = function resetTimers() {
        // Get rid of old timeouts (if any)
        window.clearTimeout(receive_timeout);
        window.clearTimeout(unstableTimeout);
        window.clearTimeout(pingTimeout);

        // Clear unstable status
        if (tunnel.state === Guacamole.Tunnel.State.UNSTABLE) tunnel.setState(Guacamole.Tunnel.State.OPEN);

        // Set new timeout for tracking overall connection timeout
        receive_timeout = window.setTimeout(function () {
            close_tunnel(new Guacamole.Status(Guacamole.Status.Code.UPSTREAM_TIMEOUT, 'Server timeout.'));
        }, tunnel.receiveTimeout);

        // Set new timeout for tracking suspected connection instability
        unstableTimeout = window.setTimeout(function () {
            tunnel.setState(Guacamole.Tunnel.State.UNSTABLE);
        }, tunnel.unstableThreshold);

        var currentTime = new Date().getTime();
        var pingDelay = Math.max(lastSentPing + PING_FREQUENCY - currentTime, 0);

        // Ping tunnel endpoint regularly to test connection stability, sending
        // the ping immediately if enough time has already elapsed
        if (pingDelay > 0) pingTimeout = window.setTimeout(sendPing, pingDelay);
        else sendPing();
    };

    /**
     * Closes this tunnel, signaling the given status and corresponding
     * message, which will be sent to the onerror handler if the status is
     * an error status.
     *
     * @private
     * @param {!Guacamole.Status} status
     *     The status causing the connection to close;
     */
    function close_tunnel(status) {
        // Get rid of old timeouts (if any)
        window.clearTimeout(receive_timeout);
        window.clearTimeout(unstableTimeout);
        window.clearTimeout(pingTimeout);

        // Ignore if already closed
        if (tunnel.state === Guacamole.Tunnel.State.CLOSED) return;

        // If connection closed abnormally, signal error.
        if (status.code !== Guacamole.Status.Code.SUCCESS && tunnel.onerror) tunnel.onerror(status);

        // Mark as closed
        tunnel.setState(Guacamole.Tunnel.State.CLOSED);

        socket.close();
    }

    this.sendMessage = function (elements) {
        // Do not attempt to send messages if not connected
        if (!tunnel.isConnected()) return;

        // Do not attempt to send empty messages
        if (arguments.length === 0) return;

        /**
         * Converts the given value to a length/string pair for use as an
         * element in a Guacamole instruction.
         *
         * @private
         * @param {*} value
         *     The value to convert.
         *
         * @return {!string}
         *     The converted value.
         */
        function getElement(value) {
            var string = new String(value);
            return string.length + '.' + string;
        }

        // Initialized message with first element
        var message = getElement(arguments[0]);

        // Append remaining elements
        for (var i = 1; i < arguments.length; i++) message += ',' + getElement(arguments[i]);

        // Final terminator
        message += ';';

        socket.send(message);
    };

    this.connect = function (data) {
        resetTimers();

        // Mark the tunnel as connecting
        tunnel.setState(Guacamole.Tunnel.State.CONNECTING);

        // Connect socket
        socket = new WebSocket(tunnelURL + '?' + data);

        socket.onopen = function (event) {
            resetTimers();
        };

        socket.onclose = function (event) {
            // Pull status code directly from closure reason provided by Guacamole
            if (event.reason) close_tunnel(new Guacamole.Status(parseInt(event.reason), event.reason));
            // Failing that, derive a Guacamole status code from the WebSocket
            // status code provided by the browser
            else if (event.code) close_tunnel(new Guacamole.Status(Guacamole.Status.Code.fromWebSocketCode(event.code)));
            // Otherwise, assume server is unreachable
            else close_tunnel(new Guacamole.Status(Guacamole.Status.Code.UPSTREAM_NOT_FOUND));
        };

        socket.onmessage = function (event) {
            resetTimers();

            var message = event.data;
            var startIndex = 0;
            var elementEnd;

            var elements = [];

            do {
                // Search for end of length
                var lengthEnd = message.indexOf('.', startIndex);
                if (lengthEnd !== -1) {
                    // Parse length
                    var length = parseInt(message.substring(elementEnd + 1, lengthEnd));

                    // Calculate start of element
                    startIndex = lengthEnd + 1;

                    // Calculate location of element terminator
                    elementEnd = startIndex + length;
                }

                // If no period, incomplete instruction.
                else close_tunnel(new Guacamole.Status(Guacamole.Status.Code.SERVER_ERROR, 'Incomplete instruction.'));

                // We now have enough data for the element. Parse.
                var element = message.substring(startIndex, elementEnd);
                var terminator = message.substring(elementEnd, elementEnd + 1);

                // Add element to array
                elements.push(element);

                // If last element, handle instruction
                if (terminator === ';') {
                    // Get opcode
                    var opcode = elements.shift();

                    // Update state and UUID when first instruction received
                    if (tunnel.uuid === null) {
                        // Associate tunnel UUID if received
                        if (opcode === Guacamole.Tunnel.INTERNAL_DATA_OPCODE && elements.length === 1) tunnel.setUUID(elements[0]);

                        // Tunnel is now open and UUID is available
                        tunnel.setState(Guacamole.Tunnel.State.OPEN);
                    }

                    // Call instruction handler.
                    if (opcode !== Guacamole.Tunnel.INTERNAL_DATA_OPCODE && tunnel.oninstruction) tunnel.oninstruction(opcode, elements);

                    // Clear elements
                    elements.length = 0;
                }

                // Start searching for length at character after
                // element terminator
                startIndex = elementEnd + 1;
            } while (startIndex < message.length);
        };
    };

    this.disconnect = function () {
        close_tunnel(new Guacamole.Status(Guacamole.Status.Code.SUCCESS, 'Manually closed.'));
    };
};

Guacamole.WebSocketTunnel.prototype = new Guacamole.Tunnel();

/**
 * Guacamole Tunnel which cycles between all specified tunnels until
 * no tunnels are left. Another tunnel is used if an error occurs but
 * no instructions have been received. If an instruction has been
 * received, or no tunnels remain, the error is passed directly out
 * through the onerror handler (if defined).
 *
 * @constructor
 * @augments Guacamole.Tunnel
 * @param {...Guacamole.Tunnel} tunnelChain
 *     The tunnels to use, in order of priority.
 */
Guacamole.ChainedTunnel = function (tunnelChain) {
    /**
     * Reference to this chained tunnel.
     * @private
     */
    var chained_tunnel = this;

    /**
     * Data passed in via connect(), to be used for
     * wrapped calls to other tunnels' connect() functions.
     * @private
     */
    var connect_data;

    /**
     * Array of all tunnels passed to this ChainedTunnel through the
     * constructor arguments.
     * @private
     */
    var tunnels = [];

    /**
     * The tunnel committed via commit_tunnel(), if any, or null if no tunnel
     * has yet been committed.
     *
     * @private
     * @type {Guacamole.Tunnel}
     */
    var committedTunnel = null;

    // Load all tunnels into array
    for (var i = 0; i < arguments.length; i++) tunnels.push(arguments[i]);

    /**
     * Sets the current tunnel.
     *
     * @private
     * @param {!Guacamole.Tunnel} tunnel
     *     The tunnel to set as the current tunnel.
     */
    function attach(tunnel) {
        // Set own functions to tunnel's functions
        chained_tunnel.disconnect = tunnel.disconnect;
        chained_tunnel.sendMessage = tunnel.sendMessage;

        /**
         * Fails the currently-attached tunnel, attaching a new tunnel if
         * possible.
         *
         * @private
         * @param {Guacamole.Status} [status]
         *     An object representing the failure that occured in the
         *     currently-attached tunnel, if known.
         *
         * @return {Guacamole.Tunnel}
         *     The next tunnel, or null if there are no more tunnels to try or
         *     if no more tunnels should be tried.
         */
        var failTunnel = function failTunnel(status) {
            // Do not attempt to continue using next tunnel on server timeout
            if (status && status.code === Guacamole.Status.Code.UPSTREAM_TIMEOUT) {
                tunnels = [];
                return null;
            }

            // Get next tunnel
            var next_tunnel = tunnels.shift();

            // If there IS a next tunnel, try using it.
            if (next_tunnel) {
                tunnel.onerror = null;
                tunnel.oninstruction = null;
                tunnel.onstatechange = null;
                attach(next_tunnel);
            }

            return next_tunnel;
        };

        /**
         * Use the current tunnel from this point forward. Do not try any more
         * tunnels, even if the current tunnel fails.
         *
         * @private
         */
        function commit_tunnel() {
            tunnel.onstatechange = chained_tunnel.onstatechange;
            tunnel.oninstruction = chained_tunnel.oninstruction;
            tunnel.onerror = chained_tunnel.onerror;

            // Assign UUID if already known
            if (tunnel.uuid) chained_tunnel.setUUID(tunnel.uuid);

            // Assign any future received UUIDs such that they are
            // accessible from the main uuid property of the chained tunnel
            tunnel.onuuid = function uuidReceived(uuid) {
                chained_tunnel.setUUID(uuid);
            };

            committedTunnel = tunnel;
        }

        // Wrap own onstatechange within current tunnel
        tunnel.onstatechange = function (state) {
            switch (state) {
                // If open, use this tunnel from this point forward.
                case Guacamole.Tunnel.State.OPEN:
                    commit_tunnel();
                    if (chained_tunnel.onstatechange) chained_tunnel.onstatechange(state);
                    break;

                // If closed, mark failure, attempt next tunnel
                case Guacamole.Tunnel.State.CLOSED:
                    if (!failTunnel() && chained_tunnel.onstatechange) chained_tunnel.onstatechange(state);
                    break;
            }
        };

        // Wrap own oninstruction within current tunnel
        tunnel.oninstruction = function (opcode, elements) {
            // Accept current tunnel
            commit_tunnel();

            // Invoke handler
            if (chained_tunnel.oninstruction) chained_tunnel.oninstruction(opcode, elements);
        };

        // Attach next tunnel on error
        tunnel.onerror = function (status) {
            // Mark failure, attempt next tunnel
            if (!failTunnel(status) && chained_tunnel.onerror) chained_tunnel.onerror(status);
        };

        // Attempt connection
        tunnel.connect(connect_data);
    }

    this.connect = function (data) {
        // Remember connect data
        connect_data = data;

        // Get committed tunnel if exists or the first tunnel on the list
        var next_tunnel = committedTunnel ? committedTunnel : tunnels.shift();

        // Attach first tunnel
        if (next_tunnel) attach(next_tunnel);
        // If there IS no first tunnel, error
        else if (chained_tunnel.onerror) chained_tunnel.onerror(Guacamole.Status.Code.SERVER_ERROR, 'No tunnels to try.');
    };
};

Guacamole.ChainedTunnel.prototype = new Guacamole.Tunnel();

/**
 * Guacamole Tunnel which replays a Guacamole protocol dump from a static file
 * received via HTTP. Instructions within the file are parsed and handled as
 * quickly as possible, while the file is being downloaded.
 *
 * @constructor
 * @augments Guacamole.Tunnel
 * @param {!string} url
 *     The URL of a Guacamole protocol dump.
 *
 * @param {boolean} [crossDomain=false]
 *     Whether tunnel requests will be cross-domain, and thus must use CORS
 *     mechanisms and headers. By default, it is assumed that tunnel requests
 *     will be made to the same domain.
 *
 * @param {object} [extraTunnelHeaders={}]
 *     Key value pairs containing the header names and values of any additional
 *     headers to be sent in tunnel requests. By default, no extra headers will
 *     be added.
 */
Guacamole.StaticHTTPTunnel = function StaticHTTPTunnel(url, crossDomain, extraTunnelHeaders) {
    /**
     * Reference to this Guacamole.StaticHTTPTunnel.
     *
     * @private
     */
    var tunnel = this;

    /**
     * AbortController instance which allows the current, in-progress HTTP
     * request to be aborted. If no request is currently in progress, this will
     * be null.
     *
     * @private
     * @type {AbortController}
     */
    var abortController = null;

    /**
     * Additional headers to be sent in tunnel requests. This dictionary can be
     * populated with key/value header pairs to pass information such as authentication
     * tokens, etc.
     *
     * @private
     * @type {!object}
     */
    var extraHeaders = extraTunnelHeaders || {};

    /**
     * The number of bytes in the file being downloaded, or null if this is not
     * known.
     *
     * @type {number}
     */
    this.size = null;

    this.sendMessage = function sendMessage(elements) {
        // Do nothing
    };

    this.connect = function connect(data) {
        // Ensure any existing connection is killed
        tunnel.disconnect();

        // Connection is now starting
        tunnel.setState(Guacamole.Tunnel.State.CONNECTING);

        // Create Guacamole protocol and UTF-8 parsers specifically for this
        // connection
        var parser = new Guacamole.Parser();
        var utf8Parser = new Guacamole.UTF8Parser();

        // Invoke tunnel's oninstruction handler for each parsed instruction
        parser.oninstruction = function instructionReceived(opcode, args) {
            if (tunnel.oninstruction) tunnel.oninstruction(opcode, args);
        };

        // Allow new request to be aborted
        abortController = new AbortController();

        // Stream using the Fetch API
        fetch(url, {
            headers: extraHeaders,
            credentials: crossDomain ? 'include' : 'same-origin',
            signal: abortController.signal,
        }).then(function gotResponse(response) {
            // Reset state and close upon error
            if (!response.ok) {
                if (tunnel.onerror) tunnel.onerror(new Guacamole.Status(Guacamole.Status.Code.fromHTTPCode(response.status), response.statusText));

                tunnel.disconnect();
                return;
            }

            // Report overall size of stream in bytes, if known
            tunnel.size = response.headers.get('Content-Length');

            // Connection is open
            tunnel.setState(Guacamole.Tunnel.State.OPEN);

            var reader = response.body.getReader();
            var processReceivedText = function processReceivedText(result) {
                // Clean up and close when done
                if (result.done) {
                    tunnel.disconnect();
                    return;
                }

                // Parse only the portion of data which is newly received
                parser.receive(utf8Parser.decode(result.value));

                // Continue parsing when next chunk is received
                reader.read().then(processReceivedText);
            };

            // Schedule parse of first chunk
            reader.read().then(processReceivedText);
        });
    };

    this.disconnect = function disconnect() {
        // Abort any in-progress request
        if (abortController) {
            abortController.abort();
            abortController = null;
        }

        // Connection is now closed
        tunnel.setState(Guacamole.Tunnel.State.CLOSED);
    };
};

Guacamole.StaticHTTPTunnel.prototype = new Guacamole.Tunnel();

/**
 * Parser that decodes UTF-8 text from a series of provided ArrayBuffers.
 * Multi-byte characters that continue from one buffer to the next are handled
 * correctly.
 *
 * @constructor
 */
Guacamole.UTF8Parser = function UTF8Parser() {
    /**
     * The number of bytes remaining for the current codepoint.
     *
     * @private
     * @type {!number}
     */
    var bytesRemaining = 0;

    /**
     * The current codepoint value, as calculated from bytes read so far.
     *
     * @private
     * @type {!number}
     */
    var codepoint = 0;

    /**
     * Decodes the given UTF-8 data into a Unicode string, returning a string
     * containing all complete UTF-8 characters within the provided data. The
     * data may end in the middle of a multi-byte character, in which case the
     * complete character will be returned from a later call to decode() after
     * enough bytes have been provided.
     *
     * @private
     * @param {!ArrayBuffer} buffer
     *     Arbitrary UTF-8 data.
     *
     * @return {!string}
     *     The decoded Unicode string.
     */
    this.decode = function decode(buffer) {
        var text = '';

        var bytes = new Uint8Array(buffer);
        for (var i = 0; i < bytes.length; i++) {
            // Get current byte
            var value = bytes[i];

            // Start new codepoint if nothing yet read
            if (bytesRemaining === 0) {
                // 1 byte (0xxxxxxx)
                if ((value | 0x7f) === 0x7f) text += String.fromCharCode(value);
                // 2 byte (110xxxxx)
                else if ((value | 0x1f) === 0xdf) {
                    codepoint = value & 0x1f;
                    bytesRemaining = 1;
                }

                // 3 byte (1110xxxx)
                else if ((value | 0x0f) === 0xef) {
                    codepoint = value & 0x0f;
                    bytesRemaining = 2;
                }

                // 4 byte (11110xxx)
                else if ((value | 0x07) === 0xf7) {
                    codepoint = value & 0x07;
                    bytesRemaining = 3;
                }

                // Invalid byte
                else text += '\uFFFD';
            }

            // Continue existing codepoint (10xxxxxx)
            else if ((value | 0x3f) === 0xbf) {
                codepoint = (codepoint << 6) | (value & 0x3f);
                bytesRemaining--;

                // Write codepoint if finished
                if (bytesRemaining === 0) text += String.fromCharCode(codepoint);
            }

            // Invalid byte
            else {
                bytesRemaining = 0;
                text += '\uFFFD';
            }
        }

        return text;
    };
};

/**
 * The unique ID of this version of the Guacamole JavaScript API. This ID will
 * be the version string of the guacamole-common-js Maven project, and can be
 * used in downstream applications as a sanity check that the proper version
 * of the APIs is being used (in case an older version is cached, for example).
 *
 * @type {!string}
 */
Guacamole.API_VERSION = '1.5.0';

/**
 * Abstract video player which accepts, queues and plays back arbitrary video
 * data. It is up to implementations of this class to provide some means of
 * handling a provided Guacamole.InputStream and rendering the received data to
 * the provided Guacamole.Display.VisibleLayer. Data received along the
 * provided stream is to be played back immediately.
 *
 * @constructor
 */
Guacamole.VideoPlayer = function VideoPlayer() {
    /**
     * Notifies this Guacamole.VideoPlayer that all video up to the current
     * point in time has been given via the underlying stream, and that any
     * difference in time between queued video data and the current time can be
     * considered latency.
     */
    this.sync = function sync() {
        // Default implementation - do nothing
    };
};

/**
 * Determines whether the given mimetype is supported by any built-in
 * implementation of Guacamole.VideoPlayer, and thus will be properly handled
 * by Guacamole.VideoPlayer.getInstance().
 *
 * @param {!string} mimetype
 *     The mimetype to check.
 *
 * @returns {!boolean}
 *     true if the given mimetype is supported by any built-in
 *     Guacamole.VideoPlayer, false otherwise.
 */
Guacamole.VideoPlayer.isSupportedType = function isSupportedType(mimetype) {
    // There are currently no built-in video players (and therefore no
    // supported types)
    return false;
};

/**
 * Returns a list of all mimetypes supported by any built-in
 * Guacamole.VideoPlayer, in rough order of priority. Beware that only the core
 * mimetypes themselves will be listed. Any mimetype parameters, even required
 * ones, will not be included in the list.
 *
 * @returns {!string[]}
 *     A list of all mimetypes supported by any built-in Guacamole.VideoPlayer,
 *     excluding any parameters.
 */
Guacamole.VideoPlayer.getSupportedTypes = function getSupportedTypes() {
    // There are currently no built-in video players (and therefore no
    // supported types)
    return [];
};

/**
 * Returns an instance of Guacamole.VideoPlayer providing support for the given
 * video format. If support for the given video format is not available, null
 * is returned.
 *
 * @param {!Guacamole.InputStream} stream
 *     The Guacamole.InputStream to read video data from.
 *
 * @param {!Guacamole.Display.VisibleLayer} layer
 *     The destination layer in which this Guacamole.VideoPlayer should play
 *     the received video data.
 *
 * @param {!string} mimetype
 *     The mimetype of the video data in the provided stream.
 *
 * @return {Guacamole.VideoPlayer}
 *     A Guacamole.VideoPlayer instance supporting the given mimetype and
 *     reading from the given stream, or null if support for the given mimetype
 *     is absent.
 */
Guacamole.VideoPlayer.getInstance = function getInstance(stream, layer, mimetype) {
    // There are currently no built-in video players
    return null;
};
export default Guacamole;
