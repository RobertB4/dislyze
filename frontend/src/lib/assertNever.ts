export function assertNever(value: never): never {
	throw new Error(`Expected never, received unexpected value,: ${value}`);
}
