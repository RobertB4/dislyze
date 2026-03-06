export class KnownError {
	constructor(private _message: string) {}

	get message(): string {
		return this._message;
	}
}
