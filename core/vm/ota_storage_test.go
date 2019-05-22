// Copyright 2018 Wanchain Foundation Ltd

package vm

import (
	"bytes"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"math/big"
	"testing"
)

var (
	otaShortAddrs = []string{
		"0x022c849aefd10287bb1fb831524a83403ecefc9d546fbf73ef5e95b79c3cb5ae7602ca02565436af262a4cc9197145278d355aee79140e201e35879c5ac72f5dbd2f",
		"0x0348cc8f64f14085eb24e100db9dbd46d217a44451c571f3ebbb8a1b387e2a613c03ef64a43cc2f4498a6641dcee5afe317654d72f61971c03821a1f1b06a32a58db",
		"0x02864c100e06bcfc53ad86aecd0d14b126bc90268b5a64e267556244281d7c0288032f82c8055f947a1509885f5551804fcfb6fa084c2b0915a286747a892cdaba54",
		"0x03bfdf88c14bda519d7d348be2b3a04e9ea7888e064707ffd9bba9dc264e6d8c9f03d7ea3d3d10f39115ff00c70606cae16e9ef7dbcb533f907d3d05e88983023e5e",
		"0x02850cbb0c4b8e3930e5dd79eb7b736c38e24514f89168f87a25496658713a90a4029eccc7471db606ed4a279b4571e4a4ea2f0158ebf53e20071c85d0b2d1ec5fab",
		"0x02483128152168625de2b21b4d7ba1f8e98a160ea78361b3225695517385fc3218023f1f8f4079be98200f882cbdaabbc6cc18ceae48b44f6bf9053de09d024de9be",
		"0x0305246565268865843190a09ece7cce28c9295d11f79930ca1787f2f044e413fd02e87f1b3c3103f028a000c7bda3e09d82f56e63ac1edf157f8955c61a059aa8a8",
		"0x027037ad331a3028d9005f1eb2b78b288fcece677c380142ea5b9919f1302ed00b032a5e555c0bbb29c42b5f5e7402f35bc22bc34d0d008dac41b00ad43fdb39f6d5",
		"0x039d89050b5981bcb6de8c47cdee5365b8676698cba82ccc244cea33ff4da814d6026c94b7fa6b5ce6bb67449d2db032271081abb1dde056de4a2f31130a979e9479",
	}

	otaMixSetAddrs = []string{
		"0345677dd5c14406945bd22e5b03af2db518690649523877aac5d3438cde4234370354a50db18be990a93c245883271e5875263e5cc614aecef0ccd2e6cecdae46d6",
		"02296efc8293f6d4b488a687a07f16e9716d09ccf226e5d6132297c5fbcc732edd034344965cee2dd786831c5d2887ad62b6cc6363c53df7e61023503e5e22b01ff1",
		"02613f9ea642a4e17bc59c6b806776534b3e4d1fa9d06b977ba0e31003fc6b775802a3b0c239cd195292f110212bdef9cd2f4a87f7146ebdaa37f07155b78fd8aa32",
		"0275c5c0632ee0961f4e3575e5f4bf70bbc0d4e87752672d9cbbaf17892c38e1ff026b6eda7eb9dee482188f139359fd02db6632b75f7e5abed400df9bebad67cf05",
		"024e4f0e02a1306cbb84051717538ca372c540111c388f0a9e1a37bb51da14e5b502e8dc38ae31f0f762d20f61cf2c6197a85a4c7771141e614b8b866892061708aa",
		"023f684bb0b634d265a60a64e3c0acb58527b710ea47dd438ba4e9bb684654fb9703369e4cb790612e5ef16ef0993f91e62ee9cec63c802b02ab8923b129b4b4f33b",
		"026cc75d1d2b0bbfaa6f88092e2c440860b501a50060ed41afe70ae21c6609e4c503420c38f7107bd674e6f08c6d055c979e5c4b12add96fa160cde72fab41e091d0",
		"035f4996d104565103438ee5ade3ba34dad12a4035e2ba7232b75ebd711ec203830313404fadcb529e9a3729cde2b3ee4da9dc6920389a03c0954235d786f240244d",
		"02443b23d45855fe44add2252786f329f973d85d268b44dc9a8c49caecda249c9203102b240b49d5a4351d5497beac94b6e7b18f532ad8cf25e0f7cbdcf522ca80e4",
		"027767754af7cbd056fd3da7e4684d378244d6476910e5a175a1d8f66b396c4c72030f8d056dd1926cf5dbb0761c2c8209b0564948d7274bfd62083f936dd0b30360",
		"03bcc152a969015950efdc6caa9fe3da7fafa554d2da20d873224857b5defe65740271ef5117002351c79bdf7ceba0b2dc3e5fcbc19201dfd75e52d43ae171481c64",
		"03031a913d6d3d65b20ea782c083838776515e8339778cb27b0554a8c2f507926f0223930620c9340e3b407469f0214c7ef424216c94b631d5ca45c2fecd84bf6fc4",
		"03e38622e56e8d1481fccdd9b10f089e738c1654f479252e88251459a26a49485e02536ff9c6b860e9067acf3df277f0a88832848910dba90eacb0caccd26db2fcec",
		"02026cbbdfa8e285a723dc7ab96b0fc59a6bc90b00d3dba2c7e49e70735f434b0702acdf7017c148e457cc3df1e1344743bf973d72c7a07a5e90bed34875f0c457af",
		"0292b449f4b95b1365f586304b0d812bf9f89d3969bb871d4cfd2442d434c1408b03326b9d547559d8193105097718f1b6104375059281f32d1499b697be5e1ecbef",
		"03a2389ad24bb2fdad626dcfb5c708ab6f6488e49f32a429c3cb5585e897b20fd5021fca8ea16f7ddbbeb599cb52a7992c0f2cf27d464dee882061b1f174ff46518f",
		"0369028a62bc80e3ab5c32e4462b6b2436f535718967730f92a7b4d8c692a3543d0306ea0188604a287a5076a4c8f80681141301b1f6b9ae62c1a2b448853d1595b3",
		"02b62e18d6765426b10d5678b668996305b9e71bef0143a6c487a0abd0ade2551c02f9b1cb397dbf98045fca68b222a4c9ba3bc50591eda80c77ed60748ad6099b0f",
		"0253e4190bc7000fb94e72551c3c995c66c29773b0925d54829843a8b0cf74fdd5025877aff7ffb92dd2aba4e0a39d1a72450168e8d2ca0e0e7ec47b6c9933f06f16",
		"03eedb35d5f3680fa6e535bd166b28e5ebdb1aac444b31f92b6a30cff1bf0b826503152f3f221a4df963503a8dddaba122c5574602307e6f8ca8627d948478a2ffa3",
		"035424c0fa7725680a40798dc1e848a6d5419092f30beab9bd7101f607376446e202aab80e8a1a3b6a84192074b64a29d7eb4ac416c4b18dfe3ac5f5c53ffbb5d344",
		"02246c1e5807b503c869555c72c8e0696eeadeda94ddf2a916dd44cf09f9248be7027b785216c5c803e076e0b16c32278fd02ab3fe0b4d86368a791099339aa23dff",
		"0301a8dc84e6994fd73bed91603ae9db80592a1548e0338526171a31f2593a2a6d03ac85f09660dff31180bdf60db3ce7821474a8a2a1f596946f27419e2435ea665",
		"02b3c0ccdd78b2bb7658fba741580ffa269b76b1066403883758cca640827ef17703f6a86fab023c81a6919ef7b5655f2b99aa5af9fceaa706ebfb3c9cc417bcdc44",
		"03d12c9ea6c92079606941a38bea695182fd9bb1a97c52df67fdaac4e20549c1f203a2696274405b6ca045548b21685dd30b86b01caaaf9acd8d0f07c0895ba93010",
		"036f10d5f5a72f1eb7022dee80f73c56a3d98762695fd8d4d9215179192887fdd1029c51c2699c858eb96d42172dd80002b2b2f88a177aa8f5713190c76a2dd7dcd1",
		"02f493719e11903da6ac39e64c9ec669fe49221ebb42091c10c150d4b097842dfa0323cf1a6f3d6e7458c335b066510de4105d3bd1930b070b9fb95a8980f1309c95",
		"0294b5df5a3473fad47a83ba5160198f618a18f7aa16945cc8a6fe6fb826f6c2a8038a06f5d02cf4da43806f8d1451db84dbe63e139c4329a872105a97c6c61b311c",
		"039db56bc204f1631e768766f4f329657b4961c8db6011685ac193dfcc6ecafb8202143b3398c1b203dad209ae4fde8272c6ff2ea0203d364390d5c6c6605c15ccd8",
		"02d5158e0841b6a8c1f90fa5847194654d7a805369d41a5411a762625995bc2a6a0346c2000172bfa0f2d34248b1375cc30e25724b3728041b31791d101ab757cb2b",
		"02db10e1f27001f60fe4fd9335b1f8f0f0ff35c2cac1466de3203f743a6a0c000f027af6c98f4af8d89db22d87d5c6f9c1c96350518e193f1531d629577baef26b3e",
		"03a62717a7e0ee6563550fa42c3075bad24bcbc4e696c3c57216faa2d678c70cb803086dd9fcf1972952bfcf67ba96fb022cdc59f55dd61045a7e79f61ab7259dfee",
		"03ba22fd5c70faa32654b4727d717ecc705bbaebd4418d327b149c1b26c3cf089b022c0ad4422dfd354c291a5205b14d6574711abbef376f6e0bd666fc848fc136c6",
		"02a302740b26c7cc60eb4b9dbd0a772e4bb99f9681b7527f980282d25a3e6f228203d20a8ff046d214b4a717f21bad6fd412654dc574846e1531aac50aabd4030682",
		"02b3c5f09e8adc7b3bd767d3fec7b149395e3a8a0a02460d869ec1b5c9e0c22a8a036d6b496d371790a7a9e0739fa85e29a9dfef05235d14ac888d1d3b04e983b8d1",
		"03d2d155adb26c7a69cde34a98a1a3e90c345b19ef053afdf130f6157e7b5c849202c5c1a6a3fab02e3e68b6045102b552f6dde13c734f7a833838cf821d6e848792",
		"02d310d351eb694001b9cd170c65a48a0afa9b0d3d40910af0da9fc97c8232051e027ff42b3f0a76deff74c06d0f31c8f3fe2489f7af05e7194ef5daf39057bc0672",
		"0228678d008cf51fd67a8409c1bb6a83832673bcb6bda98a0b3b481c2db59f32ae03045ce1f57c0ee72277f47605d9340bec9bf00a0c01504c8a942b225320460f02",
		"02808cc9ef0f20fa1dd88d3b49c95e03a5d5eb372c42ad0ef25a01363530a725a10364af6b5ae595dcca7ffb3c70b6f9ef6a8891396c018304c7beb6d0ffde494c9f",
		"0246321b86689fdb44522328c80ff260bd37411dad5a595cb06b6a78c0094afb4c03f00592ca8dd745491a0e0b40d686dd4c029bf820d22126d6ea8982b854055db9",
		"032f39386e45e009f8cc6872bfc485e4dd2e9e276f18366485c00b2a1d13df0b3a036a98241a67e24efb2d18c0eaf7fc89b3e9f78a36dc5b1683b014efa185314761",
		"03f39fe9d263c664a1f1eacdf5d0718c75b25b520b881f13ed2dd583cd8edcc88302a3623c3c72e44b39545d898a3d4fa61d100985a0de513ffd3dd86152875c2cb7",
		"02e3266ad9cad4a53cb2b49cf384ea6dac649d6595f6410a4b13e7b42e5346273003600ca1862a4b42ad2f70fc30cbbe477f7c1c15cf9dab07209b3fe1db4c0e5035",
		"030043f261253b63a61235be84daf1b9a976d5b4bd4de885fece6d8a185517bd400359c81205db387a185172bc4f7293a23a7a2ea4f37bb04a36f22ab381aec97198",
		"03cc230e682c1a1758f755be2ea5c513e1b5f23d8e29aff0f5c9c87f4e247f78f002fc32f147d6229e5561c20ea7837391b6aa1f6b98d333027f65687658b8dd0d9c",
		"02bce836983d53579aa9d9318090443e79d46ae59439349efac40801685f78ec3f022515ed0945198e136908e8eb4bb4433e2b69ff388c1815786d842117c025c055",
		"02e5c98c4cdc9e3e1e3a76aff69378c7715b0f3c22f6af71b06d878ce6730e8b5d030f184ffa5c1f190fea374a13a8d867dc337e1bf23fb73d8740c2b972705a31fc",
		"03189cff12f5e82d840de1177b65f748f67cad7490222b54b23879c80a153829ca03987dbc75cf17c04c73e7b03489a103fafb350ab633051e52403c9c82ec625547",
		"03204de59afcca1fe6cc7e546077be8a989ae0c3ce4b137255dc34438f6b6072ed0203ed4c5a4b13d03ff55e9c76ddaf8a8ca33958e8ff5b620325b81a36005de39c",
		"02ec5e3c4035f4ddf14a05b6b01d8ae978b8b5322316016012e50f62d94e12b508027ba6c309c302696d39b3c301a9af7a332240019871c733410ab0e2f0471b4030",
		"0331cd68127f683467c971bfe6f580d0ed7c3983b261c2e637d10e02e5552a40d103edc7839a03fecca40d1d8c832d70a94e23b4bf905769b4390aab4aa184fbd349",
		"02fdd74baf355caa20c803e20e6aab9f116f9f07352c028a39fc9817a31e7d283703ec758b11ba083133c939b8faab78162ac21371d56ba394a65fb464c2cf70dfe7",
		"02d0e74c5a576e9644ff16f1268ac17ae4940cfab8a91401bc86e69da52170769f035b6441f07bf9c05bf4e26c070f106abbb49ccae2cafcc16d236ff2facca16685",
		"031a46331b0c0c6eb5a378eaedf063be498048a1a15e50c397752dd078d7ab1dc7032bad37342169ee779c3110972a0b77d2907f0d148f8b4e803644696861f0c4d2",
		"0259b0fbf936bea8c46c2d2e5065b49da0812aaf6546b2d294bcfccdfb0f37bbb203897b491179291210050925cb4c0f99830c84688f5476a9aae0d2ef71ec0af42c",
		"02a681409a2ca8ffcc054abdce551b8c5e8fc49384119f1881eab8c6851073386e03ef40d9fbf4e0f5d5a21f213e1faa1b2ca5eb321a94da1a41abbf9d6d6da03b7f",
		"03062d24a85db1b9346c7b4b5fed7f58f98943a69402f55825d4f39cfe3e2e6eaa027c934d8a458bdd5476b16998f33e2980a4f048997c22239be9902c401d6bdbf3",
		"027c89e76924902afdac486d9d3f9b7677679cfce06d33e419cfc3132ae63bcff1021cb2d8bb2ca809fea308a2c1d18f74cefb8dd4002d3316c732fc824a4de89f0b",
		"022f4d6174ae3f35f779c5d62fb1a3735ab073a12a9dafecf4514eb058032bb10c03b03c0f83f127ce1b31b06a99d7f6ee0f13bbf16bd47f86ce70dcddda4ab3a418",
		"0351929f675e6e3a8c31256e553c57ca0ff15a49a191f409e7eff3e58fc3f3ef3003076fde3fabc12367e5a79f8e01a2fd7ee3addc6568c14e8e8adddfb36a0be59c",
		"0356fbe24ffbeab869d3c4973e03765b55050719f70cc189f6ec1731a5271943b1038d4db476dd5545b54220f3da53dc20e8b10cbf74b24a1bb72ba28b1231e8dfcd",
		"03ff9bcbb7bb824fc459d4ba2f4a8ec203e5c0c5bfa35eed9a42871a7c8ec8b136037e3f2f4576657b63d32b6df207e184e649af8d19c59eedd4bf2d381d00c8d34c",
		"0256b94edf885ae93756b52b9e90dfc0cc368982cf43907faabc42572b795d784c03ec82ef2d6c47e6b8a9117f8497c4ff24fbf29c68f1e16d9e6e18af0acc512cba",
		"0214544f30858022be597d9a6135155a897e419aae590ea6c241480ec8817d7e5a03028ececa6308092f8a6fda2e4a170422618c2f02fb78ce105cacecde7c99b166",
		"031ea7b51d749e07da3f85f0f44746ff5b059197d8514d2a9204a02e7355734e6f02203b307d63f87c5edf5292a82886c745f3640f2883afb4af6d03139a4fbb8940",
		"02ea5cdbf7b40275f822de5a124f749b73dfcbb7305e28e5d168a93f2abff51b6f031fabb5ce9a75d7ff264f0665df232fe705f6fc233c20d4ab1fae353714046295",
		"03e72ddc3add58c0ce2926f9c45c2173e3e960e10cd6f4db36c5a4e23221890ce003d0527941902ac1e84138ebc631a3cd2643031f61c5fc8b7443eeb92a913bed90",
		"02598d8cf21adbeb7d838e5eaa565fabcd1a23a0602f0f881387e991dbda15304d02e679601f6b95ca5557b0754961e4e94f0ca047e11e76b60deafede3c5b6334a6",
		"03d818f4d359d7868b1c23f2a1c182dd75a66753fa50226957b54cc8e049ae78810274abc42ab5dd5a925ad3130d45eac887a962c275a119cbe5940d8b00846ab1ad",
		"033350f1f023949fe863b7039b030eb27e819f074fcf7cab729a11a805575ff00f02e9070e812301301ca4d411ce2c8850f8b69bf343f3becd54b5284f6f4f2aebdb",
		"0207d3f70f99c68c693c436a9aaa10585ab15efe7f752489386c9dc1df6f45c70c035dc2f9a2ae8431f937d3de0aae7cecd7f6a3ce689fdb53ffdd684c240b32c5d8",
		"030a02107f4e105465347bbf87ab4994708f75430eae979eb6343e6b93bdc9c3160286ffb56634a3dcb1543f194ee9cec2799fb77f93699b8e98992ad8670958aaf8",
		"0200d6e7fd6c5f907fd586c0da75f2d0da408be183957f6eef5433ed24307a781d038d366fb426c2d78ea18195195a96f0d9ae25bf641b874a26bdf5600109a2db68",
		"03b105e4f4ea87408dfa183f322b35f85b8f0272e913ee04109341911840ec731d02b47a64f2a9228e0846e66679f518e5260e94309188c6f7d12173b3e4003ed04c",
		"025884e9eee012d2642da839d09f0a1f0d3d03ac398a0c30ec790e2e2c1c0226ef0329b7cb3590d2bbbbb24753c01965ad193f23c4eb204d7f83ee9b7b822666045f",
		"028bced3ed46d83c5d8a73695d9ee21b699e3162c0eddefa73ad573c74b291c7ab02b6b6a263f2180fdf00cba8f9c9c2c211c90c993c1d3325ca14c6c434a66ba4e0",
		"02b4b41b2bc6a5177debc6aaead1a4259582d92b289d08c19b00c94078f4d09953031d9b6003d8124b6ea586d36ac1092c8e1a18b729440adad40163d300390500c8",
		"03363c202ab84582cb7c5d48613581c713796bdde2ba9f8f3ca086ff7cb46dc3a5036ffd7507d3c7452683352c106d71ff94f0a4dabc14ceaa86e7bc257a8362912e",
		"02101d266a899ce1d120752bf9be105693c098c6f1561c5af204a04556ad15b75003b5cec201bfc00364479d8f8572f74aaf6fb9b049b29d063c30a05590200d9e2a",
		"034ba0172a11e0bf3b04d2200b4ab2fa1f55e51d2f404e30f697a34a604304f381024a7340e5297c8e280cb78000bc6caf7d53371534d2a64554027ee76f606fcd14",
		"0244e4ef42951845073c7bbdc14f1087ef3c790efc0690ca1f2fd0a6eada524652031ca519d19ad62b8e4ac6dbf6bb44543d7c565c673982640bea32e02dea6edad9",
		"0207584426460218a04ae021b187dc689f7966e190ac4717e3c9f9db8e9b0dee21035d3aad0ae9c379294b91126f3496ee675779b8978b524dd506dd256156d4d368",
		"026d7b49ab33327724652f03d127775e06e413ea6f9908045ad01a80d07bfdf7c203e3c6f4b6a60fd6661ac561254e1e11a61bf0300a37a41428f7b82368b76ae296",
		"02fb177eb8779cbac2e78711433413b3057ad4c20cf4c059d3145f56889a1887410274f7909057f4f1aa964b00a1e29695bc16d69135f2d3c1705ff9d2b0b6d1d657",
		"0262a2a03160b9c8cae206eb11bb8d11d38f6b2b6c7df2ed102e52decb597ccd7a02833ff4593e48788fe145cabe0c604b6638efd1645fd0bfc1fe5e0a62b08d65a5",
		"02a5600908dbaffc21d476e1f525e5615e35e1201f3a303dc4aab0f1bb50fd8dfd0376d9f9d8fbfd281f264e40ac1a1e06b015445e2e068931111a878eaf14322d12",
		"03f783adfb17ff6a3159e4064803e7b243c8cf6d1cf450ba7677b30bf78e339cac026d80012216c4b287d2c324370217ddb55ebae09f12d6762db9e8270c388e1a46",
		"03f27ab925a2df5bca077db2e7b707ae02da1fc321acf48ee93b439c5d972e620d0360920390ff3c054aad64b9335ae2c3553a2b12e27dcc9289c1ae546c7e25e470",
	}
)

func TestOTABalance2ContractAddr(t *testing.T) {

	{
		addr := OTABalance2ContractAddr(nil)
		if (addr != common.Address{}) {
			t.Error("expect empoty address!")
		}
	}

	{
		addr := OTABalance2ContractAddr(big.NewInt(10))
		if (addr != common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16}) {
			t.Error("unexpect address!")
		}
	}

}

func TestGetOtaBalance(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaShortAddr = common.FromHex(otaShortAddrs[0])
	)

	t.Logf("otaShortAddr len:%d", len(otaShortAddr))
	otaAX := otaShortAddr[1:common.HashLength]
	balance, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err == nil || balance != nil {
		t.Errorf("otaAX len:%d, err:%s", len(otaAX), err.Error())
	}

	otaAX = otaShortAddr[1 : 1+common.HashLength]
	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if balance != nil && balance.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("balance:%v", balance)
	}

	err = setOTA(statedb, big.NewInt(10), otaShortAddr)
	if err != nil {
		t.Errorf("SetOTA err:%s", err.Error())
		return
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("GetOtaBalanceFromAX err:%s", err.Error())
	}

	if balance == nil || balance.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("GetOtaBalanceFromAX balance:%v", balance)
	}
}

func TestCheckOTAExist(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaShortAddr = common.FromHex(otaShortAddrs[1])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	_, _, err := CheckOTAAXExist(nil, otaAX)
	if err == nil {
		t.Error("expect err: invalid input param!")
	}

	_, _, err = CheckOTAAXExist(statedb, otaAX[1:])
	if err == nil {
		t.Error("expect err: invalid input param!")
	}

	exist, balanceGet, err := CheckOTAAXExist(statedb, otaAX)
	if err != nil {
		t.Errorf("CheckOTAExist, err:%s", err.Error())
	}

	if exist || (balanceGet != nil && balanceGet.Cmp(common.Big0) != 0) {
		t.Errorf("exis:%t, balance:%v", exist, balanceGet)
	}

	err = setOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("SetOTA err:%s", err.Error())
	}

	exist, balanceGet, err = CheckOTAAXExist(statedb, otaAX)
	if err != nil {
		t.Errorf("CheckOTAExist, err:%s", err.Error())
	}
	if !exist || balanceGet == nil || balanceGet.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("ChechOTAExist, exis:%t, balanceGet:%v", exist, balanceGet)
	}
}

func TestBatCheckOTAExist(t *testing.T) {

	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaShortAddrBytes = [][]byte{
				common.FromHex(otaShortAddrs[1]),
				common.FromHex(otaShortAddrs[2]),
				common.FromHex(otaShortAddrs[3]),
				common.FromHex(otaShortAddrs[4]),
			}
		)

		otaAXs := make([][]byte, 0, 4)
		for _, otaShortAddr := range otaShortAddrBytes {
			otaAXs = append(otaAXs, otaShortAddr[0:1+common.HashLength])
		}

		_, _, _, err := BatCheckOTAExist(nil, otaAXs)
		if err == nil {
			t.Error("expect err: invalid input param!")
		}

		_, _, _, err = BatCheckOTAExist(statedb, nil)
		if err == nil {
			t.Error("expect err: invalid input param!")
		}

		otaAXs = append(otaAXs, otaAXs[0][1:])
		_, _, _, err = BatCheckOTAExist(statedb, nil)
		if err == nil {
			t.Error("expect err: invalid input ota AX!")
		}

	}

	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaShortAddrBytes = [][]byte{
				common.FromHex(otaShortAddrs[1]),
				common.FromHex(otaShortAddrs[2]),
				common.FromHex(otaShortAddrs[3]),
				common.FromHex(otaShortAddrs[4]),
			}

			balanceSet = big.NewInt(10)
		)

		otaAXs := make([][]byte, 0, 4)
		for _, otaShortAddr := range otaShortAddrBytes {
			otaAXs = append(otaAXs, otaShortAddr[0:1+common.HashLength])
		}

		exist, balanceGet, unexisotaAx, err := BatCheckOTAExist(statedb, otaAXs)
		if exist || (balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0) {
			t.Errorf("exis:%t, balanceGet:%v", exist, balanceGet)
		}

		if unexisotaAx == nil {
			t.Errorf("unexisotaAX is nil!")
		}

		if common.ToHex(unexisotaAx) != common.ToHex(otaAXs[0]) {
			t.Errorf("unexisotaAx:%s, expect:%s", common.ToHex(unexisotaAx), common.ToHex(otaAXs[0]))
		}

		if err != nil {
			t.Logf("err:%s", err.Error())
		}

		for _, otaShortAddr := range otaShortAddrBytes {
			err = setOTA(statedb, balanceSet, otaShortAddr)
			if err != nil {
				t.Errorf("err:%s", err.Error())
			}
		}

		exist, balanceGet, unexisotaAx, err = BatCheckOTAExist(statedb, otaAXs)
		if !exist || (balanceGet != nil && balanceSet.Cmp(balanceGet) != 0) {
			t.Errorf("exis:%t, balanceGet:%v", exist, balanceGet)
		}

		if unexisotaAx != nil {
			t.Errorf("unexisota:%s", common.ToHex(unexisotaAx))
		}

		if err != nil {
			t.Errorf("err:%s", err.Error())
		}

		unexisotaShortAddr := common.FromHex(otaShortAddrs[5])
		unexisotaAXSet := unexisotaShortAddr[0 : 1+common.HashLength]
		otaAXs = append(otaAXs, unexisotaAXSet)
		exist, balanceGet, unexisotaAx, err = BatCheckOTAExist(statedb, otaAXs)
		if exist || (balanceGet != nil && balanceSet.Cmp(balanceGet) == 0) {
			t.Errorf("exis:%t, balanceGet:%v", exist, balanceGet)
		}

		if unexisotaAx != nil {
			t.Logf("unexisotaAx:%s", common.ToHex(unexisotaAx))
		}
		if err != nil {
			t.Logf("err:%s", err.Error())
		}

		err = setOTA(statedb, big.NewInt(0).Add(balanceSet, big.NewInt(10)), unexisotaShortAddr)
		if err != nil {
			t.Errorf("err:%s", err.Error())
		}

		exist, balanceGet, unexisotaAx, err = BatCheckOTAExist(statedb, otaAXs)
		if exist || (balanceGet != nil && balanceSet.Cmp(balanceGet) == 0) {
			t.Errorf("exis:%t, balanceGet:%v", exist, balanceGet)
		}

		if exist || (balanceGet != nil && balanceSet.Cmp(balanceGet) == 0) {
			t.Errorf("exis:%t, balanceGet:%v", exist, balanceGet)
		}

		if err != nil {
			t.Logf("err:%s", err.Error())
		}

		if unexisotaAx == nil {
			t.Errorf("unexisota is nil!")
		}

		if common.ToHex(unexisotaAx) != common.ToHex(unexisotaAXSet) {
			t.Errorf("unexisota:%s, expect:%s", common.ToHex(unexisotaAx), common.ToHex(unexisotaAXSet))
		}
	}

}

func TestSetOTA(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaShortAddr = common.FromHex(otaShortAddrs[3])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	t.Logf("otaShortAddr len:%d", len(otaShortAddr))

	err := setOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	balance, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if balance == nil || balance.Cmp(balanceSet) != 0 {
		t.Errorf("balance:%v", balance)
	}
}

func TestAddOTAIfNotExist(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaShortAddr = common.FromHex(otaShortAddrs[4])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	add, err := AddOTAIfNotExist(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if !add {
		t.Errorf("add is false!")
	}

	add, err = AddOTAIfNotExist(statedb, balanceSet, otaShortAddr)
	if err == nil {
		t.Errorf("expect err: ota exist already!")
	}

	if add {
		t.Errorf("add is true!")
	}

	balance, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if balance == nil || balance.Cmp(balanceSet) != 0 {
		t.Errorf("balance:%v", balance)
	}
}

func TestSetOtaBalanceToAX(t *testing.T) {
	{
		err := SetOtaBalanceToAX(nil, make([]byte, common.HashLength), big1)
		if err == nil {
			t.Error("expect err: invalid input param!")
		}
	}

	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
		)

		err := SetOtaBalanceToAX(statedb, make([]byte, common.HashLength-1), big1)
		if err == nil {
			t.Error("expect err: invalid input param!")
		}
	}

	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
		)

		err := SetOtaBalanceToAX(statedb, make([]byte, common.HashLength), nil)
		if err == nil {
			t.Error("expect err: invalid input param!")
		}
	}
}

func TestGetOTAInfoFromAX(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaShortAddr = common.FromHex(otaShortAddrs[4])
		otaAX        = otaShortAddr[0 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	otaShortAddrGet, balanceGet, err := GetOTAInfoFromAX(statedb, otaAX)
	if otaShortAddrGet != nil {
		t.Errorf("otaShortAddrGet is not nil.")
	}

	if balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("balance is not 0! balance:%s", balanceGet.String())
	}

	if err == nil {
		t.Errorf("err is nil!")
	}

	err = setOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	otaShortAddrGet, balanceGet, err = GetOTAInfoFromAX(statedb, otaAX)
	if otaShortAddrGet == nil {
		t.Errorf("otaShortAddrGet is nil!")
	}

	if common.ToHex(otaShortAddrGet) != common.ToHex(otaShortAddr) {
		t.Errorf("otaShortAddrGet:%s, expect:%s", common.ToHex(otaShortAddrGet), common.ToHex(otaShortAddr))
	}

	if balanceGet == nil {
		t.Errorf("balanceGet is nil!")
	}

	if balanceSet.Cmp(balanceGet) != 0 {
		t.Errorf("balanceGet:%v, expect:%v", balanceGet, balanceSet)
	}

}

//func TestGetOTASet(t *testing.T) {
//	{
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr = common.FromHex(otaShortAddrs[6])
//			otaAX      = otaWanAddr[1 : 1+common.HashLength]
//		)
//
//		setLen := 3
//		_, _, err := GetOTASet(statedb, otaAX, setLen)
//		if err == nil {
//			t.Error("err is nil! expect err: can't find ota address balance!")
//		}
//	}
//
//	{
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr = common.FromHex(otaShortAddrs[6])
//			otaAX      = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet = big.NewInt(10)
//
//			setLen = 3
//		)
//
//		err := SetOtaBalanceToAX(statedb, otaAX, balanceSet)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		_, _, err = GetOTASet(statedb, otaAX, setLen)
//		if err == nil {
//			t.Error("err is nil! expect err: no ota address exist! balance:10")
//		}
//
//	}
//
//	for i := 0; i < 100; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr = common.FromHex(otaShortAddrs[6])
//			otaAX      = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet = big.NewInt(10)
//
//			setLen = 1
//		)
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		for _, otaGet := range otaSet {
//			if !bytes.Equal(otaGet, otaWanAddr) {
//				t.Error("ota addr in set is wrong! expect:", common.ToHex(otaWanAddr), ", actual:", common.ToHex(otaGet))
//			}
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 100; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr = common.FromHex(otaShortAddrs[6])
//			otaAX      = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet = big.NewInt(10)
//
//			setLen = 2
//		)
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		for _, otaGet := range otaSet {
//			if !bytes.Equal(otaGet, otaWanAddr) {
//				t.Error("ota addr in set is wrong! expect:", common.ToHex(otaWanAddr), ", actual:", common.ToHex(otaGet))
//			}
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 100; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr = common.FromHex(otaShortAddrs[6])
//			otaAX      = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet = big.NewInt(10)
//
//			setLen = 3
//		)
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		for _, otaGet := range otaSet {
//			if !bytes.Equal(otaGet, otaWanAddr) {
//				t.Error("ota addr in set is wrong! expect:", common.ToHex(otaWanAddr), ", actual:", common.ToHex(otaGet))
//			}
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 100; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr         = common.FromHex(otaShortAddrs[6])
//			otaMixSetAddrBytes = make([][]byte, 0, 100)
//			otaAX              = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet         = big.NewInt(10)
//
//			setLen = 1
//		)
//
//		for _, otaWanAddr := range otaMixSetAddrs {
//			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
//		}
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		err = setOTA(statedb, balanceSet, otaMixSetAddrBytes[0])
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		for _, otaGet := range otaSet {
//			if !bytes.Equal(otaGet, otaMixSetAddrBytes[0]) {
//				t.Error("ota addr in set is wrong! expect:", common.ToHex(otaMixSetAddrBytes[0]), ", actual:", common.ToHex(otaGet))
//			}
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 30; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr         = common.FromHex(otaShortAddrs[6])
//			otaMixSetAddrBytes = make([][]byte, 0, 100)
//			otaAX              = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet         = big.NewInt(10)
//
//			setLen = 2
//		)
//
//		for _, otaWanAddr := range otaMixSetAddrs {
//			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
//		}
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		err = setOTA(statedb, balanceSet, otaMixSetAddrBytes[0])
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		var otaGetAX [common.HashLength]byte
//		otaAXMap := make(map[[common.HashLength]byte]bool)
//		for _, otaGet := range otaSet {
//			AXGet, _ := GetAXFromWanAddr(otaGet)
//			copy(otaGetAX[:], AXGet)
//			otaAXMap[otaGetAX] = true
//		}
//
//		if len(otaAXMap) != 2 {
//			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
//		}
//
//		copy(otaGetAX[:], otaAX)
//		_, ok := otaAXMap[otaGetAX]
//		if !ok {
//			t.Error("otaSet wrong, don't contain self!")
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 30; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr         = common.FromHex(otaShortAddrs[6])
//			otaMixSetAddrBytes = make([][]byte, 0, 100)
//			otaAX              = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet         = big.NewInt(10)
//
//			setLen = 3
//		)
//
//		for _, otaWanAddr := range otaMixSetAddrs {
//			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
//		}
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		err = setOTA(statedb, balanceSet, otaMixSetAddrBytes[0])
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		var otaGetAX [common.HashLength]byte
//		otaAXMap := make(map[[common.HashLength]byte]bool)
//		for _, otaGet := range otaSet {
//			AXGet, _ := GetAXFromWanAddr(otaGet)
//			copy(otaGetAX[:], AXGet)
//			otaAXMap[otaGetAX] = true
//		}
//
//		if len(otaAXMap) != 2 {
//			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
//		}
//
//		copy(otaGetAX[:], otaAX)
//		_, ok := otaAXMap[otaGetAX]
//		if !ok {
//			t.Error("otaSet wrong, don't contain self!")
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 30; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr         = common.FromHex(otaShortAddrs[6])
//			otaMixSetAddrBytes = make([][]byte, 0, 100)
//			otaAX              = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet         = big.NewInt(10)
//
//			setLen = 10
//		)
//
//		for _, otaWanAddr := range otaMixSetAddrs {
//			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
//		}
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		err = setOTA(statedb, balanceSet, otaMixSetAddrBytes[0])
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		var otaGetAX [common.HashLength]byte
//		otaAXMap := make(map[[common.HashLength]byte]bool)
//		for _, otaGet := range otaSet {
//			AXGet, _ := GetAXFromWanAddr(otaGet)
//			copy(otaGetAX[:], AXGet)
//			otaAXMap[otaGetAX] = true
//		}
//
//		if len(otaAXMap) != 2 {
//			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
//		}
//
//		copy(otaGetAX[:], otaAX)
//		_, ok := otaAXMap[otaGetAX]
//		if !ok {
//			t.Error("otaSet wrong, don't contain self!")
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	for i := 0; i < 30; i++ {
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr         = common.FromHex(otaShortAddrs[6])
//			otaMixSetAddrBytes = make([][]byte, 0, 100)
//			otaAX              = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet         = big.NewInt(10)
//
//			setLen = 10
//		)
//
//		for _, otaWanAddr := range otaMixSetAddrs {
//			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
//		}
//
//		err := setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Error("set ota balance fail. err:", err.Error())
//		}
//
//		for _, addrByte := range otaMixSetAddrBytes {
//			err = setOTA(statedb, balanceSet, addrByte)
//			if err != nil {
//				t.Error("set ota balance fail. err:", err.Error())
//			}
//		}
//
//		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Error("get ota set fail! err: ", err.Error())
//		}
//
//		if otaSet == nil {
//			t.Error("otaSet is nil")
//		}
//
//		if len(otaSet) != setLen {
//			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
//		}
//
//		var otaGetAX [common.HashLength]byte
//		otaAXMap := make(map[[common.HashLength]byte]bool)
//		for _, otaGet := range otaSet {
//			AXGet, _ := GetAXFromWanAddr(otaGet)
//			copy(otaGetAX[:], AXGet)
//			otaAXMap[otaGetAX] = true
//		}
//
//		if len(otaAXMap) != setLen {
//			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
//		}
//
//		copy(otaGetAX[:], otaAX)
//		_, ok := otaAXMap[otaGetAX]
//		if ok {
//			t.Error("otaSet wrong, contain self!")
//		}
//
//		if balanceGet == nil {
//			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
//		}
//
//		if balanceSet.Cmp(balanceGet) != 0 {
//			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
//		}
//
//	}
//
//	{
//		var (
//			db, _      = ethdb.NewMemDatabase()
//			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
//
//			otaWanAddr         = common.FromHex(otaShortAddrs[6])
//			otaMixSetAddrBytes = make([][]byte, 0, 100)
//			otaAX              = otaWanAddr[1 : 1+common.HashLength]
//			balanceSet         = big.NewInt(10)
//		)
//
//		for _, otaWanAddr := range otaMixSetAddrs {
//			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
//		}
//
//		setLen := 3
//		otaShortAddrBytesGet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
//		if err == nil {
//			t.Errorf("err is nil!")
//		}
//
//		if otaShortAddrBytesGet != nil {
//			t.Errorf("otaShortAddrBytesGet is not nil!")
//		}
//
//		if balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0 {
//			t.Errorf("balanceGet is not 0! balanceGet:%s", balanceGet.String())
//		}
//
//		err = setOTA(statedb, balanceSet, otaWanAddr)
//		if err != nil {
//			t.Errorf("err:%s", err.Error())
//		}
//
//		for _, otaShortAddrTmp := range otaMixSetAddrBytes {
//			err = setOTA(statedb, balanceSet, otaShortAddrTmp)
//			if err != nil {
//				t.Errorf("err:%s", err.Error())
//			}
//		}
//
//		// mem database Iterator doesnt work. unit test alwayse fail!!
//		otaShortAddrBytesGet, balanceGet, err = GetOTASet(statedb, otaAX, setLen)
//		if err != nil {
//			t.Errorf("err:%s", err.Error())
//		}
//
//		if otaShortAddrBytesGet == nil {
//			t.Errorf("otaShortAddrBytesGet is nil!")
//		}
//
//		if len(otaShortAddrBytesGet) != setLen {
//			t.Errorf("otaShortAddrBytesGet len is wrong! len:%d, expect:%d", len(otaShortAddrBytesGet), setLen)
//		}
//
//		for _, otaShortAddrGet := range otaShortAddrBytesGet {
//			otaAXGet := otaShortAddrGet[1 : 1+common.HashLength]
//			otaShortAddrReGet, balanceReGet, err := GetOTAInfoFromAX(statedb, otaAXGet)
//			if err != nil {
//				t.Errorf("err:%s", err.Error())
//			}
//
//			if common.ToHex(otaShortAddrReGet) != common.ToHex(otaShortAddrGet) {
//				t.Errorf("otaShortAddrReGet:%s, expect:%s", common.ToHex(otaShortAddrReGet), common.ToHex(otaShortAddrGet))
//			}
//
//			if balanceReGet == nil {
//				t.Errorf("balanceReGet is nil!")
//			}
//
//			if balanceReGet.Cmp(balanceSet) != 0 {
//				t.Errorf("balanceReGet:%s, expect:%s", balanceReGet.String(), balanceSet.String())
//			}
//		}
//
//	}
//}

func TestGetOTASet(t *testing.T) {
	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
		)

		setLen := 3
		_, _, err := GetOTASet(statedb, otaAX, setLen)
		expectErr := "can't find ota address balance!"
		if err.Error() != expectErr {
			t.Error("err is nil! expect err: ", expectErr)
		}
	}

	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 3
		)

		err := SetOtaBalanceToAX(statedb, otaAX, balanceSet)
		if err != nil {
			t.Error("set ota balance fail. err:", err.Error())
		}

		_, _, err = GetOTASet(statedb, otaAX, setLen)
		expectErr := "no ota exist! balance:10"
		if err.Error() != expectErr {
			t.Error("err is nil! expect err: no ota exist! balance:10")
		}
	}

	{
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 1
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		if err != nil {
			t.Error("set ota balance fail. err:", err.Error())
		}

		_, _, err = GetOTASet(statedb, otaAX, setLen)
		expectErr := "too more required ota number! balance:10, exist count:1"
		if err.Error() != expectErr {
			t.Error("get ota set fail! err: ", err.Error(), ", expected:", expectErr)
		}
	}

	for i := 0; i < 100; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 1
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[7]))

		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
		if err != nil {
			t.Error("get ota set fail! err: ", err.Error())
		}

		if otaSet == nil {
			t.Error("otaSet is nil")
		}

		if len(otaSet) != setLen {
			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
		}

		if !bytes.Equal(otaSet[0], common.FromHex(otaShortAddrs[7])) {
			t.Error("otaSet value wrong!, contain unexpected ota")
		}

		if balanceGet == nil {
			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
		}

		if balanceSet.Cmp(balanceGet) != 0 {
			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
		}
	}

	for i := 0; i < 10; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 2
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[7]))

		_, _, err = GetOTASet(statedb, otaAX, setLen)
		expectErr := "too more required ota number! balance:10, exist count:2"
		if err.Error() != expectErr {
			t.Error("get ota set fail! err: ", err.Error(), ", expected:", expectErr)
		}
	}

	for i := 0; i < 100; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 1
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[7]))
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[8]))

		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
		if err != nil {
			t.Error("get ota set fail! err: ", err.Error())
		}

		if otaSet == nil {
			t.Error("otaSet is nil")
		}

		if len(otaSet) != setLen {
			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
		}

		if bytes.Equal(otaSet[0], otaWanAddr) {
			t.Error("otaSet value wrong!, contain unexpected ota")
		}

		if balanceGet == nil {
			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
		}

		if balanceSet.Cmp(balanceGet) != 0 {
			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
		}
	}

	for i := 0; i < 100; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 2
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[7]))
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[8]))

		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
		if err != nil {
			t.Error("get ota set fail! err: ", err.Error())
		}

		if otaSet == nil {
			t.Error("otaSet is nil")
		}

		if len(otaSet) != setLen {
			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
		}

		var otaGetAX [common.HashLength]byte
		otaAXMap := make(map[[common.HashLength]byte]bool)
		for _, otaGet := range otaSet {
			if bytes.Equal(otaGet, otaWanAddr) {
				t.Error("otaSet value wrong!, contain unexpected ota")
			}

			AXGet, _ := GetAXFromWanAddr(otaGet)
			copy(otaGetAX[:], AXGet)
			otaAXMap[otaGetAX] = true
		}

		if len(otaAXMap) != setLen {
			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
		}

		if balanceGet == nil {
			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
		}

		if balanceSet.Cmp(balanceGet) != 0 {
			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
		}
	}

	for i := 0; i < 10; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 3
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[7]))
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[8]))

		_, _, err = GetOTASet(statedb, otaAX, setLen)
		expectErr := "too more required ota number! balance:10, exist count:3"
		if err.Error() != expectErr {
			t.Error("get ota set fail! err: ", err.Error(), ", expected:", expectErr)
		}
	}

	for i := 0; i < 10; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr = common.FromHex(otaShortAddrs[6])
			otaAX      = otaWanAddr[1 : 1+common.HashLength]
			balanceSet = big.NewInt(10)

			setLen = 4
		)

		err := setOTA(statedb, balanceSet, otaWanAddr)
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[7]))
		err = setOTA(statedb, balanceSet, common.FromHex(otaShortAddrs[8]))

		_, _, err = GetOTASet(statedb, otaAX, setLen)
		expectErr := "too more required ota number! balance:10, exist count:3"
		if err.Error() != expectErr {
			t.Error("get ota set fail! err: ", err.Error(), ", expected:", expectErr)
		}
	}

	for i := 0; i < 100; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr         = common.FromHex(otaShortAddrs[6])
			otaMixSetAddrBytes = make([][]byte, 0, 100)
			otaAX              = otaWanAddr[1 : 1+common.HashLength]
			balanceSet         = big.NewInt(10)

			setLen = 10
		)

		for _, otaWanAddr := range otaMixSetAddrs {
			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
		}

		err := setOTA(statedb, balanceSet, otaWanAddr)
		if err != nil {
			t.Error("set ota balance fail. err:", err.Error())
		}

		for _, addrByte := range otaMixSetAddrBytes {
			err = setOTA(statedb, balanceSet, addrByte)
			if err != nil {
				t.Error("set ota balance fail. err:", err.Error())
			}
		}

		otaSet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
		if err != nil {
			t.Error("get ota set fail! err: ", err.Error())
		}

		if otaSet == nil {
			t.Error("otaSet is nil")
		}

		if len(otaSet) != setLen {
			t.Error("otaSet len wrong! expect:", setLen, ", actual:", len(otaSet))
		}

		var otaGetAX [common.HashLength]byte
		otaAXMap := make(map[[common.HashLength]byte]bool)
		for _, otaGet := range otaSet {
			AXGet, _ := GetAXFromWanAddr(otaGet)
			copy(otaGetAX[:], AXGet)
			otaAXMap[otaGetAX] = true
		}

		if len(otaAXMap) != setLen {
			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
		}

		copy(otaGetAX[:], otaAX)
		_, ok := otaAXMap[otaGetAX]
		if ok {
			t.Error("otaSet wrong, contain self!")
		}

		if balanceGet == nil {
			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
		}

		if balanceSet.Cmp(balanceGet) != 0 {
			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
		}
	}

	for i := 0; i < 30; i++ {
		var (
			db, _      = ethdb.NewMemDatabase()
			statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

			otaWanAddr         = common.FromHex(otaShortAddrs[6])
			otaMixSetAddrBytes = make([][]byte, 0, 100)
			otaAX              = otaWanAddr[1 : 1+common.HashLength]
			balanceSet         = big.NewInt(10)
		)

		for _, otaWanAddr := range otaMixSetAddrs {
			otaMixSetAddrBytes = append(otaMixSetAddrBytes, common.FromHex(otaWanAddr))
		}

		setLen := 3
		otaShortAddrBytesGet, balanceGet, err := GetOTASet(statedb, otaAX, setLen)
		expectErr := "can't find ota address balance!"
		if err.Error() != expectErr {
			t.Error("err is nil! expect err: ", expectErr)
		}

		if otaShortAddrBytesGet != nil {
			t.Errorf("otaShortAddrBytesGet is not nil!")
		}

		if balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0 {
			t.Errorf("balanceGet is not 0! balanceGet:%s", balanceGet.String())
		}

		err = setOTA(statedb, balanceSet, otaWanAddr)
		if err != nil {
			t.Errorf("err:%s", err.Error())
		}

		for _, otaShortAddrTmp := range otaMixSetAddrBytes {
			err = setOTA(statedb, balanceSet, otaShortAddrTmp)
			if err != nil {
				t.Errorf("err:%s", err.Error())
			}
		}

		// mem database Iterator doesnt work. unit test alwayse fail!!
		otaShortAddrBytesGet, balanceGet, err = GetOTASet(statedb, otaAX, setLen)
		if err != nil {
			t.Errorf("err:%s", err.Error())
		}

		if otaShortAddrBytesGet == nil {
			t.Errorf("otaShortAddrBytesGet is nil!")
		}

		if len(otaShortAddrBytesGet) != setLen {
			t.Errorf("otaShortAddrBytesGet len is wrong! len:%d, expect:%d", len(otaShortAddrBytesGet), setLen)
		}

		var otaGetAX [common.HashLength]byte
		otaAXMap := make(map[[common.HashLength]byte]bool)
		for _, otaGet := range otaShortAddrBytesGet {
			AXGet, _ := GetAXFromWanAddr(otaGet)
			copy(otaGetAX[:], AXGet)
			otaAXMap[otaGetAX] = true
		}

		if len(otaAXMap) != setLen {
			t.Error("otaSet's non repeating ele is wrong. expect: ", setLen, ", actual:", len(otaAXMap))
		}

		copy(otaGetAX[:], otaAX)
		_, ok := otaAXMap[otaGetAX]
		if ok {
			t.Error("otaSet wrong, contain self!")
		}

		if balanceGet == nil {
			t.Error("balance from GetOTASet is nil! expect:", balanceSet.Uint64())
		}

		if balanceSet.Cmp(balanceGet) != 0 {
			t.Error("balance from GetOTASet is nul! expect:", balanceSet.Uint64(), ", actual:", balanceGet.Uint64())
		}

		for _, otaShortAddrGet := range otaShortAddrBytesGet {
			otaAXGet := otaShortAddrGet[1 : 1+common.HashLength]
			otaShortAddrReGet, balanceReGet, err := GetOTAInfoFromAX(statedb, otaAXGet)
			if err != nil {
				t.Errorf("err:%s", err.Error())
			}

			if !bytes.Equal(otaShortAddrReGet, otaShortAddrGet) {
				t.Errorf("otaShortAddrReGet:%s, expect:%s", common.ToHex(otaShortAddrReGet), common.ToHex(otaShortAddrGet))
			}

			if balanceReGet == nil {
				t.Errorf("balanceReGet is nil!")
			}

			if balanceReGet.Cmp(balanceSet) != 0 {
				t.Errorf("balanceReGet:%s, expect:%s", balanceReGet.String(), balanceSet.String())
			}
		}
	}
}

func TestCheckOTAImageExist(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaWanAddr = common.FromHex(otaShortAddrs[7])
		balanceSet = big.NewInt(10)
	)

	otaImage := crypto.Keccak256(otaWanAddr)
	otaImageValue := balanceSet.Bytes()

	exist, otaImageValueGet, err := CheckOTAImageExist(statedb, otaImage)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if exist {
		t.Errorf("exist is true!")
	}

	if otaImageValueGet != nil && len(otaImageValueGet) != 0 {
		t.Errorf("otaImageValueGet is not empoty!")
	}

	err = AddOTAImage(statedb, otaImage, otaImageValue)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	exist, otaImageValueGet, err = CheckOTAImageExist(statedb, otaImage)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if !exist {
		t.Errorf("exist is false!")
	}

	if otaImageValueGet == nil || common.ToHex(otaImageValueGet) != common.ToHex(otaImageValue) {
		t.Errorf("otaImageValueGet:%s, expect:%s", common.ToHex(otaImageValueGet), common.ToHex(otaImageValue))
	}
}

func TestAddOTAImage(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))

		otaWanAddr = common.FromHex(otaShortAddrs[7])
		balanceSet = big.NewInt(10)
	)

	otaImage := crypto.Keccak256(otaWanAddr)
	otaImageValue := balanceSet.Bytes()

	err := AddOTAImage(statedb, otaImage, otaImageValue)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	err = AddOTAImage(statedb, otaImage, otaImageValue)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}
}
