export namespace config {
	
	export class Emulator {
	    path: string;
	    shouldNotify: boolean;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Emulator(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.shouldNotify = source["shouldNotify"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class CfgFile {
	    emulators: Emulator[];
	
	    static createFrom(source: any = {}) {
	        return new CfgFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.emulators = this.convertValues(source["emulators"], Emulator);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

