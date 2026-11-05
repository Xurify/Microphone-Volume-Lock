export namespace main {
	
	export class State {
	    level: number;
	    target: number;
	    locked: boolean;
	    available: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.target = source["target"];
	        this.locked = source["locked"];
	        this.available = source["available"];
	        this.message = source["message"];
	    }
	}

}

