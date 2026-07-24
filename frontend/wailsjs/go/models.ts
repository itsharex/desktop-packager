export namespace main {
	
	export class ProxyRule {
	    path: string;
	    target: string;
	    rewrite: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxyRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.target = source["target"];
	        this.rewrite = source["rewrite"];
	        this.enabled = source["enabled"];
	    }
	}
	export class BuildConfig {
	    appName: string;
	    iconPath: string;
	    distPath: string;
	    tempPath: string;
	    proxyRules: ProxyRule[];
	    windowWidth: number;
	    windowHeight: number;
	    windowFullscreen: boolean;
	    windowMaximized: boolean;
	    confirmClose: boolean;
	    version: string;
	    description: string;
	    company: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appName = source["appName"];
	        this.iconPath = source["iconPath"];
	        this.distPath = source["distPath"];
	        this.tempPath = source["tempPath"];
	        this.proxyRules = this.convertValues(source["proxyRules"], ProxyRule);
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.windowFullscreen = source["windowFullscreen"];
	        this.windowMaximized = source["windowMaximized"];
	        this.confirmClose = source["confirmClose"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.company = source["company"];
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
	export class DistInfo {
	    path: string;
	    fileCount: number;
	    totalSize: number;
	    valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DistInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileCount = source["fileCount"];
	        this.totalSize = source["totalSize"];
	        this.valid = source["valid"];
	    }
	}

}

