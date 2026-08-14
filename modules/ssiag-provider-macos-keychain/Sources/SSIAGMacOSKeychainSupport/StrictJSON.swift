import CryptoKit
import Foundation

enum StrictJSONError: Error {
    case malformed
    case duplicateKey
    case invalidTopLevel
}

func strictJSONObject(_ data: Data) throws -> [String: Any] {
    var scanner = JSONStructureScanner(Array(data))
    try scanner.scanDocument()
    let value: Any
    do {
        value = try JSONSerialization.jsonObject(with: data, options: [])
    } catch {
        throw StrictJSONError.malformed
    }
    guard let object = value as? [String: Any] else {
        throw StrictJSONError.invalidTopLevel
    }
    return object
}

func canonicalJSONData(_ value: Any) throws -> Data {
    var output = Data()
    try appendCanonical(value, to: &output)
    return output
}

func canonicalDigest(_ object: [String: Any], omitting key: String) throws -> String {
    var value = object
    value.removeValue(forKey: key)
    return sha256Digest(try canonicalJSONData(value))
}

func sha256Digest(_ data: Data) -> String {
    "sha256:" + SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
}

func sha256Digest(of url: URL) throws -> String {
    let handle = try FileHandle(forReadingFrom: url)
    defer { try? handle.close() }
    var hasher = SHA256()
    while let chunk = try handle.read(upToCount: 65_536), !chunk.isEmpty {
        hasher.update(data: chunk)
    }
    return "sha256:" + hasher.finalize().map { String(format: "%02x", $0) }.joined()
}

private func appendCanonical(_ value: Any, to output: inout Data) throws {
    switch value {
    case is NSNull:
        output.append(contentsOf: "null".utf8)
    case let string as String:
        output.append(contentsOf: goQuoted(string).utf8)
    case let array as [Any]:
        output.append(0x5b)
        for index in array.indices {
            if index != array.startIndex { output.append(0x2c) }
            try appendCanonical(array[index], to: &output)
        }
        output.append(0x5d)
    case let dictionary as [String: Any]:
        output.append(0x7b)
        for (index, key) in dictionary.keys.sorted().enumerated() {
            if index != 0 { output.append(0x2c) }
            output.append(contentsOf: goQuoted(key).utf8)
            output.append(0x3a)
            guard let nested = dictionary[key] else { throw StrictJSONError.malformed }
            try appendCanonical(nested, to: &output)
        }
        output.append(0x7d)
    case let number as NSNumber:
        if CFGetTypeID(number) == CFBooleanGetTypeID() {
            output.append(contentsOf: (number.boolValue ? "true" : "false").utf8)
        } else {
            let rendered = number.stringValue
            guard rendered.range(of: #"^-?(0|[1-9][0-9]*)$"#, options: .regularExpression) != nil else {
                throw StrictJSONError.malformed
            }
            output.append(contentsOf: rendered.utf8)
        }
    default:
        throw StrictJSONError.malformed
    }
}

private func goQuoted(_ value: String) -> String {
    var output = "\""
    for scalar in value.unicodeScalars {
        switch scalar.value {
        case 0x22: output += "\\\""
        case 0x5c: output += "\\\\"
        case 0x08: output += "\\b"
        case 0x0c: output += "\\f"
        case 0x0a: output += "\\n"
        case 0x0d: output += "\\r"
        case 0x09: output += "\\t"
        case 0x00...0x1f, 0x3c, 0x3e, 0x26, 0x2028, 0x2029:
            if scalar.value <= 0xffff {
                output += String(format: "\\u%04x", scalar.value)
            } else {
                let adjusted = scalar.value - 0x10000
                output += String(format: "\\u%04x\\u%04x", 0xd800 + adjusted / 0x400, 0xdc00 + adjusted % 0x400)
            }
        default:
            output.unicodeScalars.append(scalar)
        }
    }
    output += "\""
    return output
}

private struct JSONStructureScanner {
    private let bytes: [UInt8]
    private var index = 0
    private var depth = 0

    init(_ bytes: [UInt8]) {
        self.bytes = bytes
    }

    mutating func scanDocument() throws {
        skipWhitespace()
        try scanValue()
        skipWhitespace()
        guard index == bytes.count else { throw StrictJSONError.malformed }
    }

    private mutating func scanValue() throws {
        guard index < bytes.count, depth < 64 else { throw StrictJSONError.malformed }
        switch bytes[index] {
        case 0x7b: try scanObject()
        case 0x5b: try scanArray()
        case 0x22: _ = try scanString()
        case 0x74: try scanLiteral("true")
        case 0x66: try scanLiteral("false")
        case 0x6e: try scanLiteral("null")
        case 0x2d, 0x30...0x39: try scanNumber()
        default: throw StrictJSONError.malformed
        }
    }

    private mutating func scanObject() throws {
        depth += 1
        defer { depth -= 1 }
        index += 1
        skipWhitespace()
        if consume(0x7d) { return }
        var keys = Set<String>()
        while true {
            guard index < bytes.count, bytes[index] == 0x22 else { throw StrictJSONError.malformed }
            let key = try scanString()
            guard keys.insert(key).inserted else { throw StrictJSONError.duplicateKey }
            skipWhitespace()
            guard consume(0x3a) else { throw StrictJSONError.malformed }
            skipWhitespace()
            try scanValue()
            skipWhitespace()
            if consume(0x7d) { return }
            guard consume(0x2c) else { throw StrictJSONError.malformed }
            skipWhitespace()
        }
    }

    private mutating func scanArray() throws {
        depth += 1
        defer { depth -= 1 }
        index += 1
        skipWhitespace()
        if consume(0x5d) { return }
        while true {
            try scanValue()
            skipWhitespace()
            if consume(0x5d) { return }
            guard consume(0x2c) else { throw StrictJSONError.malformed }
            skipWhitespace()
        }
    }

    private mutating func scanString() throws -> String {
        let start = index
        guard consume(0x22) else { throw StrictJSONError.malformed }
        while index < bytes.count {
            let byte = bytes[index]
            index += 1
            if byte == 0x22 {
                let encoded = Data(bytes[start..<index])
                guard let decoded = try? JSONSerialization.jsonObject(with: encoded, options: .fragmentsAllowed) as? String else {
                    throw StrictJSONError.malformed
                }
                return decoded
            }
            if byte < 0x20 { throw StrictJSONError.malformed }
            if byte == 0x5c {
                guard index < bytes.count else { throw StrictJSONError.malformed }
                let escaped = bytes[index]
                index += 1
                if escaped == 0x75 {
                    guard index + 4 <= bytes.count,
                          bytes[index..<(index + 4)].allSatisfy({ isHex($0) }) else {
                        throw StrictJSONError.malformed
                    }
                    index += 4
                } else if ![0x22, 0x5c, 0x2f, 0x62, 0x66, 0x6e, 0x72, 0x74].contains(escaped) {
                    throw StrictJSONError.malformed
                }
            }
        }
        throw StrictJSONError.malformed
    }

    private mutating func scanLiteral(_ literal: String) throws {
        let value = Array(literal.utf8)
        guard index + value.count <= bytes.count,
              Array(bytes[index..<(index + value.count)]) == value else {
            throw StrictJSONError.malformed
        }
        index += value.count
    }

    private mutating func scanNumber() throws {
        let start = index
        if consume(0x2d) {}
        guard index < bytes.count else { throw StrictJSONError.malformed }
        if consume(0x30) {
            if index < bytes.count, (0x30...0x39).contains(bytes[index]) { throw StrictJSONError.malformed }
        } else {
            guard (0x31...0x39).contains(bytes[index]) else { throw StrictJSONError.malformed }
            index += 1
            while index < bytes.count, (0x30...0x39).contains(bytes[index]) { index += 1 }
        }
        if consume(0x2e) {
            guard index < bytes.count, (0x30...0x39).contains(bytes[index]) else { throw StrictJSONError.malformed }
            while index < bytes.count, (0x30...0x39).contains(bytes[index]) { index += 1 }
        }
        if index < bytes.count, bytes[index] == 0x65 || bytes[index] == 0x45 {
            index += 1
            if index < bytes.count, bytes[index] == 0x2b || bytes[index] == 0x2d { index += 1 }
            guard index < bytes.count, (0x30...0x39).contains(bytes[index]) else { throw StrictJSONError.malformed }
            while index < bytes.count, (0x30...0x39).contains(bytes[index]) { index += 1 }
        }
        guard index > start else { throw StrictJSONError.malformed }
    }

    private mutating func skipWhitespace() {
        while index < bytes.count, [0x20, 0x09, 0x0a, 0x0d].contains(bytes[index]) { index += 1 }
    }

    private mutating func consume(_ byte: UInt8) -> Bool {
        guard index < bytes.count, bytes[index] == byte else { return false }
        index += 1
        return true
    }

    private func isHex(_ byte: UInt8) -> Bool {
        (0x30...0x39).contains(byte) || (0x41...0x46).contains(byte) || (0x61...0x66).contains(byte)
    }
}
