export namespace main {
	
	export class GenerateResultDTO {
	    patchPath: string;
	    modCount: number;
	    itemCount: number;
	    backupPath: string;
	    patchSummary: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.patchPath = source["patchPath"];
	        this.modCount = source["modCount"];
	        this.itemCount = source["itemCount"];
	        this.backupPath = source["backupPath"];
	        this.patchSummary = source["patchSummary"];
	    }
	}
	export class ItemDTO {
	    recordId: string;
	    sourceFile: string;
	    originalQuality: string;
	    template: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ItemDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recordId = source["recordId"];
	        this.sourceFile = source["sourceFile"];
	        this.originalQuality = source["originalQuality"];
	        this.template = source["template"];
	    }
	}
	export class ModDTO {
	    id: string;
	    name: string;
	    sourceLabel: string;
	    itemCount: number;
	    fileCount: number;
	    items: ItemDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ModDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sourceLabel = source["sourceLabel"];
	        this.itemCount = source["itemCount"];
	        this.fileCount = source["fileCount"];
	        this.items = this.convertValues(source["items"], ItemDTO);
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
	export class PathDefaultsDTO {
	    scanRoot: string;
	    outputRoot: string;

	    static createFrom(source: any = {}) {
	        return new PathDefaultsDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanRoot = source["scanRoot"];
	        this.outputRoot = source["outputRoot"];
	    }
	}
	export class ScanResultDTO {
	    scanRoot: string;
	    outputRoot: string;
	    mods: ModDTO[];
	    totalItems: number;
	    scannedFiles: number;
	    tweaksFolders: number;
	    usingVortex: boolean;
	    patchPath: string;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanRoot = source["scanRoot"];
	        this.outputRoot = source["outputRoot"];
	        this.mods = this.convertValues(source["mods"], ModDTO);
	        this.totalItems = source["totalItems"];
	        this.scannedFiles = source["scannedFiles"];
	        this.tweaksFolders = source["tweaksFolders"];
	        this.usingVortex = source["usingVortex"];
	        this.patchPath = source["patchPath"];
	        this.warnings = source["warnings"];
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
