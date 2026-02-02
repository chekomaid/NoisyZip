export namespace main {
	
	export class DecryptConfig {
	    inZip: string;
	    outZip: string;
	    compression: string;
	    encoding: string;
	    level: number;
	    strategy: string;
	    dictSize: number;
	    workers: number;
	    seed: string;
	    includeHidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DecryptConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inZip = source["inZip"];
	        this.outZip = source["outZip"];
	        this.compression = source["compression"];
	        this.encoding = source["encoding"];
	        this.level = source["level"];
	        this.strategy = source["strategy"];
	        this.dictSize = source["dictSize"];
	        this.workers = source["workers"];
	        this.seed = source["seed"];
	        this.includeHidden = source["includeHidden"];
	    }
	}
	export class DecryptResult {
	    recovered: number;
	    rebuilt: number;
	
	    static createFrom(source: any = {}) {
	        return new DecryptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recovered = source["recovered"];
	        this.rebuilt = source["rebuilt"];
	    }
	}
	export class EncryptConfig {
	    srcDir: string;
	    outZip: string;
	    compression: string;
	    encoding: string;
	    breakCDir: boolean;
	    commentSize: number;
	    fixedTime: boolean;
	    noiseFiles: number;
	    noiseSize: number;
	    level: number;
	    strategy: string;
	    dictSize: number;
	    workers: number;
	    seed: string;
	    includeHidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EncryptConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.srcDir = source["srcDir"];
	        this.outZip = source["outZip"];
	        this.compression = source["compression"];
	        this.encoding = source["encoding"];
	        this.breakCDir = source["breakCDir"];
	        this.commentSize = source["commentSize"];
	        this.fixedTime = source["fixedTime"];
	        this.noiseFiles = source["noiseFiles"];
	        this.noiseSize = source["noiseSize"];
	        this.level = source["level"];
	        this.strategy = source["strategy"];
	        this.dictSize = source["dictSize"];
	        this.workers = source["workers"];
	        this.seed = source["seed"];
	        this.includeHidden = source["includeHidden"];
	    }
	}
	export class EncryptResult {
	    total: number;
	    outZip: string;
	
	    static createFrom(source: any = {}) {
	        return new EncryptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.outZip = source["outZip"];
	    }
	}

}

