package dydx_usdfuture

import (
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"time"
)

func ParseDepth(msg []byte, depth *Depth) (err error) {
	//{"type":"subscribed","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":2,"channel":"v3_orderbook","id":"ETH-USD","contents":{"asks":[{"size":"63.452","price":"3567.5"},{"size":"135.45","price":"3567.6"},{"size":"0.3","price":"3567.9"},{"size":"2.345","price":"3568"},{"size":"1.682","price":"3568.3"},{"size":"65.689","price":"3568.6"},{"size":"0.224","price":"3568.9"},{"size":"2.395","price":"3569.2"},{"size":"1419.301","price":"3569.3"},{"size":"31.372","price":"3569.4"},{"size":"4.3","price":"3569.6"},{"size":"3.782","price":"3569.8"},{"size":"1086.46","price":"3570.1"},{"size":"2.445","price":"3570.2"},{"size":"29.031","price":"3570.7"},{"size":"1.542","price":"3571"},{"size":"3.986","price":"3571.1"},{"size":"13.59","price":"3571.5"},{"size":"6.995","price":"3571.6"},{"size":"1.602","price":"3572"},{"size":"14.9","price":"3572.3"},{"size":"2.1","price":"3572.4"},{"size":"1.89","price":"3572.5"},{"size":"3.537","price":"3572.8"},{"size":"20.991","price":"3573"},{"size":"64.642","price":"3573.5"},{"size":"0.392","price":"3573.7"},{"size":"5160.22","price":"3573.8"},{"size":"0.01","price":"3574.2"},{"size":"2.442","price":"3574.4"},{"size":"32.61","price":"3574.6"},{"size":"4.5","price":"3574.8"},{"size":"82.63","price":"3575"},{"size":"2.492","price":"3575.8"},{"size":"80.541","price":"3576"},{"size":"0.2","price":"3576.5"},{"size":"0.3","price":"3576.8"},{"size":"0.07","price":"3577.1"},{"size":"2.097","price":"3577.4"},{"size":"2.387","price":"3577.7"},{"size":"2.592","price":"3578"},{"size":"0.01","price":"3578.2"},{"size":"0.2","price":"3578.4"},{"size":"0.2","price":"3578.7"},{"size":"0.359","price":"3579.5"},{"size":"2.2","price":"3579.8"},{"size":"8","price":"3580"},{"size":"0.2","price":"3580.1"},{"size":"2.305","price":"3580.4"},{"size":"56.07","price":"3580.5"},{"size":"0.189","price":"3581.1"},{"size":"2.64","price":"3581.8"},{"size":"8.302","price":"3584"},{"size":"0.011","price":"3584.1"},{"size":"2.651","price":"3584.2"},{"size":"14.246","price":"3584.9"},{"size":"0.71","price":"3585.2"},{"size":"2.511","price":"3585.3"},{"size":"2.301","price":"3585.9"},{"size":"2.602","price":"3586.1"},{"size":"2.788","price":"3587.6"},{"size":"8","price":"3588.4"},{"size":"0.01","price":"3590.1"},{"size":"2.785","price":"3590.8"},{"size":"2.507","price":"3591"},{"size":"2.507","price":"3591.4"},{"size":"12","price":"3594"},{"size":"5","price":"3594.4"},{"size":"2.503","price":"3596.7"},{"size":"12","price":"3597"},{"size":"1","price":"3597.3"},{"size":"2","price":"3598.7"},{"size":"2","price":"3598.9"},{"size":"5","price":"3599"},{"size":"2","price":"3599.1"},{"size":"2.5","price":"3599.9"},{"size":"2.3","price":"3600"},{"size":"1","price":"3600.5"},{"size":"1","price":"3602.1"},{"size":"0.01","price":"3602.8"},{"size":"2","price":"3604"},{"size":"0.01","price":"3607"},{"size":"1.6","price":"3608.6"},{"size":"20","price":"3611"},{"size":"3","price":"3613.4"},{"size":"1","price":"3613.5"},{"size":"1","price":"3614.9"},{"size":"1","price":"3617.2"},{"size":"5.3","price":"3620"},{"size":"31","price":"3621"},{"size":"443.15","price":"3625"},{"size":"20","price":"3628"},{"size":"10","price":"3630"},{"size":"1","price":"3631"},{"size":"0.01","price":"3632.5"},{"size":"10","price":"3633"},{"size":"1.88","price":"3636"},{"size":"31.6","price":"3640"},{"size":"2.5","price":"3645"},{"size":"2","price":"3648"},{"size":"8.86","price":"3649"},{"size":"199.2","price":"3650"},{"size":"1.499","price":"3653"},{"size":"25","price":"3655"},{"size":"22","price":"3658"},{"size":"131","price":"3660"},{"size":"0.05","price":"3666"},{"size":"9","price":"3668.3"},{"size":"25","price":"3670"},{"size":"14","price":"3672"},{"size":"46.3","price":"3675"},{"size":"20","price":"3677"},{"size":"1","price":"3679"},{"size":"26","price":"3680"},{"size":"0.01","price":"3684.3"},{"size":"45","price":"3685"},{"size":"4","price":"3688"},{"size":"2","price":"3689"},{"size":"30","price":"3690"},{"size":"0.1","price":"3693"},{"size":"5","price":"3695"},{"size":"1","price":"3699"},{"size":"100.6","price":"3700"},{"size":"5","price":"3705"},{"size":"10","price":"3710"},{"size":"5","price":"3715"},{"size":"5","price":"3725"},{"size":"2","price":"3728.7"},{"size":"4","price":"3730"},{"size":"0.2","price":"3737"},{"size":"0.15","price":"3737.7"},{"size":"0.01","price":"3745"},{"size":"45.403","price":"3750"},{"size":"3","price":"3753"},{"size":"12.65","price":"3757.9"},{"size":"0.171","price":"3766"},{"size":"10.48","price":"3767.2"},{"size":"1","price":"3770"},{"size":"1","price":"3783"},{"size":"1","price":"3785"},{"size":"1","price":"3788"},{"size":"12.65","price":"3795.1"},{"size":"10","price":"3799"},{"size":"50.6","price":"3800"},{"size":"10.48","price":"3804.5"},{"size":"30","price":"3833"},{"size":"30","price":"3855"},{"size":"1","price":"3870"},{"size":"63","price":"3888"},{"size":"36.01","price":"3900"},{"size":"5","price":"3907.1"},{"size":"0.3","price":"3910"},{"size":"7.312","price":"3930"},{"size":"0.1","price":"3934"},{"size":"0.25","price":"3940"},{"size":"13","price":"3945"},{"size":"20.5","price":"3950"},{"size":"1","price":"3970"},{"size":"0.52","price":"3980"},{"size":"7.5","price":"3987"},{"size":"50.586","price":"3999"},{"size":"173.653","price":"4000"},{"size":"34.89","price":"4019.9"},{"size":"9.757","price":"4100"},{"size":"0.5","price":"4128.2"},{"size":"0.02","price":"4150"},{"size":"1.5","price":"4193"},{"size":"2","price":"4200"},{"size":"1","price":"4210"},{"size":"3.128","price":"4236"},{"size":"0.05","price":"4298"},{"size":"40.011","price":"4300"},{"size":"10","price":"4325"},{"size":"16","price":"4400"},{"size":"174.22","price":"4500"},{"size":"80","price":"4560"},{"size":"10","price":"4600"},{"size":"0.08","price":"4666"},{"size":"25","price":"4711"},{"size":"11.49","price":"4791"},{"size":"150","price":"4796.3"},{"size":"75","price":"4820"},{"size":"200","price":"4944"},{"size":"0.1","price":"4999"},{"size":"500","price":"4999.3"},{"size":"65","price":"5150"},{"size":"0.01","price":"5555"},{"size":"3","price":"6000"},{"size":"500","price":"6800"},{"size":"3","price":"6984"},{"size":"0.55","price":"6990"},{"size":"80","price":"8899.8"}],"bids":[{"size":"15","price":"3567.4"},{"size":"0.5","price":"3567.3"},{"size":"15","price":"3567"},{"size":"1.683","price":"3566.8"},{"size":"2.72","price":"3566.7"},{"size":"1.66","price":"3566.5"},{"size":"153.044","price":"3566.4"},{"size":"6.595","price":"3566.2"},{"size":"33.33","price":"3566.1"},{"size":"31.794","price":"3565.9"},{"size":"2.346","price":"3565.7"},{"size":"1100.47","price":"3565.5"},{"size":"2.397","price":"3565.3"},{"size":"29.33","price":"3565.1"},{"size":"57.631","price":"3564.8"},{"size":"4.655","price":"3564.7"},{"size":"28.58","price":"3564.1"},{"size":"15.052","price":"3564"},{"size":"21.044","price":"3563.9"},{"size":"4.252","price":"3563.8"},{"size":"50.73","price":"3563.5"},{"size":"49.114","price":"3563.1"},{"size":"1.802","price":"3563"},{"size":"2.246","price":"3562.2"},{"size":"5139.221","price":"3561.9"},{"size":"0.2","price":"3561.8"},{"size":"0.22","price":"3561.6"},{"size":"0.2","price":"3561.5"},{"size":"2.855","price":"3561.2"},{"size":"2.528","price":"3561.1"},{"size":"84.105","price":"3560.8"},{"size":"1","price":"3560.1"},{"size":"1.5","price":"3560"},{"size":"2.605","price":"3559.9"},{"size":"0.2","price":"3559.6"},{"size":"2.389","price":"3559.2"},{"size":"0.4","price":"3559"},{"size":"0.011","price":"3558.4"},{"size":"84.105","price":"3558"},{"size":"2.709","price":"3557.8"},{"size":"1","price":"3557.5"},{"size":"0.14","price":"3556.9"},{"size":"0.227","price":"3556.3"},{"size":"2.762","price":"3555.7"},{"size":"0.14","price":"3555.3"},{"size":"0.4","price":"3553"},{"size":"87.574","price":"3552.4"},{"size":"2.816","price":"3552"},{"size":"2.817","price":"3550.9"},{"size":"86","price":"3550"},{"size":"0.8","price":"3549.2"},{"size":"14.002","price":"3548.7"},{"size":"2.818","price":"3548.6"},{"size":"2.819","price":"3548.3"},{"size":"12.797","price":"3548.2"},{"size":"11","price":"3546"},{"size":"0.2","price":"3545"},{"size":"1","price":"3543.3"},{"size":"15","price":"3535"},{"size":"1.6","price":"3533.5"},{"size":"30.524","price":"3530"},{"size":"1","price":"3529.2"},{"size":"5.1","price":"3528"},{"size":"2","price":"3527"},{"size":"3","price":"3526.7"},{"size":"15","price":"3524"},{"size":"1","price":"3522"},{"size":"20","price":"3520"},{"size":"1","price":"3518.1"},{"size":"10.1","price":"3518"},{"size":"2","price":"3517"},{"size":"1","price":"3515.7"},{"size":"1","price":"3515.1"},{"size":"0.3","price":"3515"},{"size":"2","price":"3512.3"},{"size":"15","price":"3512"},{"size":"33.5","price":"3510"},{"size":"17","price":"3509"},{"size":"22","price":"3508"},{"size":"1","price":"3507"},{"size":"15","price":"3505"},{"size":"120","price":"3502"},{"size":"1","price":"3501.1"},{"size":"2.5","price":"3500.1"},{"size":"42.35","price":"3500"},{"size":"0.05","price":"3495"},{"size":"1","price":"3493.9"},{"size":"30","price":"3492"},{"size":"15","price":"3489"},{"size":"17","price":"3488"},{"size":"19","price":"3487"},{"size":"1","price":"3482"},{"size":"12","price":"3481"},{"size":"51.51","price":"3480"},{"size":"0.1","price":"3475"},{"size":"1.5","price":"3470"},{"size":"17","price":"3469"},{"size":"15","price":"3468"},{"size":"12","price":"3466"},{"size":"10","price":"3461.2"},{"size":"1","price":"3460"},{"size":"12","price":"3456"},{"size":"32","price":"3455"},{"size":"2.5","price":"3450.1"},{"size":"62","price":"3450"},{"size":"10","price":"3449"},{"size":"5","price":"3448"},{"size":"1","price":"3445"},{"size":"1","price":"3440"},{"size":"2","price":"3436"},{"size":"15","price":"3433"},{"size":"1.2","price":"3430"},{"size":"2","price":"3425"},{"size":"4","price":"3424.2"},{"size":"32","price":"3423"},{"size":"25","price":"3422"},{"size":"25","price":"3421.8"},{"size":"5","price":"3420.1"},{"size":"0.3","price":"3420"},{"size":"18","price":"3417"},{"size":"1","price":"3415"},{"size":"6.5","price":"3414"},{"size":"0.4","price":"3412"},{"size":"32","price":"3411"},{"size":"1","price":"3410"},{"size":"5","price":"3408"},{"size":"1","price":"3404"},{"size":"2","price":"3403"},{"size":"5","price":"3402.1"},{"size":"2.5","price":"3400.1"},{"size":"55.5","price":"3400"},{"size":"2","price":"3395"},{"size":"17","price":"3389"},{"size":"15","price":"3387"},{"size":"10","price":"3385"},{"size":"45","price":"3382"},{"size":"10","price":"3381.1"},{"size":"20","price":"3376"},{"size":"1","price":"3372"},{"size":"1","price":"3370"},{"size":"22","price":"3369"},{"size":"0.3","price":"3367"},{"size":"1.5","price":"3363"},{"size":"33.78","price":"3359.3"},{"size":"0.7","price":"3358"},{"size":"1","price":"3356"},{"size":"10","price":"3355"},{"size":"10","price":"3351.1"},{"size":"5","price":"3350.1"},{"size":"6.2","price":"3350"},{"size":"45.25","price":"3346"},{"size":"15","price":"3335.5"},{"size":"3.991","price":"3333"},{"size":"33.78","price":"3321.8"},{"size":"10","price":"3321.1"},{"size":"0.02","price":"3320"},{"size":"0.5","price":"3318"},{"size":"2.2","price":"3315"},{"size":"0.02","price":"3311"},{"size":"5","price":"3310"},{"size":"1.5","price":"3308.2"},{"size":"6.5","price":"3308"},{"size":"5","price":"3304.6"},{"size":"0.01","price":"3301.5"},{"size":"2.5","price":"3300.1"},{"size":"66.5","price":"3300"},{"size":"15","price":"3291.1"},{"size":"420.1","price":"3280"},{"size":"0.1","price":"3278.7"},{"size":"50","price":"3276"},{"size":"0.01","price":"3275"},{"size":"1","price":"3270"},{"size":"0.5","price":"3265"},{"size":"20","price":"3255"},{"size":"2","price":"3254"},{"size":"0.5","price":"3253"},{"size":"1","price":"3252"},{"size":"0.01","price":"3251.5"},{"size":"31.15","price":"3250"},{"size":"0.5","price":"3242.6"},{"size":"12.25","price":"3240"},{"size":"0.5","price":"3235"},{"size":"2","price":"3230"},{"size":"5","price":"3225"},{"size":"0.65","price":"3222"},{"size":"3","price":"3220"},{"size":"0.025","price":"3217"},{"size":"0.05","price":"3216.1"},{"size":"3.5","price":"3210"},{"size":"10","price":"3209"},{"size":"5","price":"3208"},{"size":"30","price":"3207"},{"size":"8","price":"3206"},{"size":"0.01","price":"3201.5"},{"size":"0.05","price":"3200.1"},{"size":"47.2","price":"3200"},{"size":"2","price":"3198"},{"size":"3","price":"3195.5"},{"size":"30","price":"3189"},{"size":"8","price":"3185"},{"size":"5","price":"3183"},{"size":"1.1","price":"3180"},{"size":"1","price":"3176"},{"size":"3","price":"3175.9"},{"size":"1","price":"3170"},{"size":"1","price":"3169"},{"size":"3","price":"3168"},{"size":"51","price":"3166"},{"size":"0.1","price":"3165.8"},{"size":"0.5","price":"3164"},{"size":"4","price":"3160"},{"size":"12","price":"3155"},{"size":"1.15","price":"3153"},{"size":"0.01","price":"3151.5"},{"size":"13","price":"3151"},{"size":"0.05","price":"3150.1"},{"size":"2.8","price":"3150"},{"size":"3","price":"3146"},{"size":"2","price":"3142"},{"size":"1.75","price":"3132"},{"size":"20","price":"3130.1"},{"size":"50","price":"3126"},{"size":"55","price":"3125"},{"size":"50","price":"3124"},{"size":"50","price":"3123"},{"size":"50.1","price":"3122"},{"size":"1","price":"3120"},{"size":"0.4","price":"3111"},{"size":"1","price":"3110"},{"size":"5.5","price":"3108"},{"size":"0.01","price":"3101.5"},{"size":"0.05","price":"3100.1"},{"size":"308.1","price":"3100"},{"size":"4","price":"3099"},{"size":"0.2","price":"3098"},{"size":"60","price":"3077"},{"size":"0.25","price":"3075"},{"size":"1.1","price":"3070"},{"size":"20","price":"3066"},{"size":"0.01","price":"3051.5"},{"size":"0.05","price":"3050.1"},{"size":"0.4","price":"3050"},{"size":"0.1","price":"3033"},{"size":"2","price":"3030"},{"size":"2","price":"3025"},{"size":"10.15","price":"3022"},{"size":"5","price":"3021"},{"size":"0.5","price":"3018"},{"size":"6","price":"3008"},{"size":"0.4","price":"3007"},{"size":"0.11","price":"3003"},{"size":"0.01","price":"3001.5"},{"size":"115","price":"3001"},{"size":"0.05","price":"3000.1"},{"size":"9.876","price":"3000"},{"size":"0.35","price":"2999"},{"size":"1","price":"2996"},{"size":"0.03","price":"2986"},{"size":"0.5","price":"2984"},{"size":"10","price":"2980"},{"size":"0.05","price":"2978"},{"size":"0.7","price":"2976"},{"size":"10","price":"2958"},{"size":"0.18","price":"2955"},{"size":"0.01","price":"2951.5"},{"size":"76.5","price":"2950"},{"size":"1","price":"2948"},{"size":"2","price":"2945"},{"size":"0.1","price":"2940"},{"size":"2","price":"2938"},{"size":"0.05","price":"2933.1"},{"size":"0.1","price":"2930"},{"size":"8.39","price":"2927"},{"size":"0.2","price":"2922"},{"size":"0.1","price":"2920"},{"size":"0.1","price":"2917.3"},{"size":"0.1","price":"2917.1"},{"size":"0.1","price":"2917"},{"size":"1","price":"2911"},{"size":"0.6","price":"2910"},{"size":"8","price":"2908"},{"size":"0.05","price":"2905"},{"size":"20","price":"2903"},{"size":"0.01","price":"2901.5"},{"size":"0.05","price":"2900.1"},{"size":"71.42","price":"2900"},{"size":"0.81","price":"2892"},{"size":"0.1","price":"2890"},{"size":"0.1","price":"2880"},{"size":"300","price":"2871"},{"size":"0.1","price":"2870"},{"size":"3","price":"2861"},{"size":"0.1","price":"2860"},{"size":"0.01","price":"2851.5"},{"size":"172.35","price":"2850"},{"size":"0.1","price":"2840"},{"size":"0.9","price":"2833"},{"size":"0.1","price":"2830"},{"size":"0.15","price":"2824.8"},{"size":"0.1","price":"2824"},{"size":"0.25","price":"2822"},{"size":"6.1","price":"2820"},{"size":"0.5","price":"2818"},{"size":"0.1","price":"2810"},{"size":"5","price":"2808"},{"size":"0.215","price":"2805.1"},{"size":"0.01","price":"2801.5"},{"size":"0.05","price":"2800.1"},{"size":"101.7","price":"2800"},{"size":"1","price":"2788"},{"size":"8","price":"2786.1"},{"size":"5","price":"2781"},{"size":"1","price":"2777"},{"size":"2","price":"2758.8"},{"size":"1","price":"2758.3"},{"size":"0.593","price":"2755"},{"size":"0.01","price":"2751.5"},{"size":"9.45","price":"2750"},{"size":"10","price":"2745"},{"size":"12","price":"2743"},{"size":"1","price":"2740"},{"size":"60","price":"2739"},{"size":"0.5","price":"2730"},{"size":"0.2","price":"2727"},{"size":"3","price":"2725"},{"size":"0.3","price":"2722"},{"size":"1","price":"2720"},{"size":"1.4","price":"2718"},{"size":"5.1","price":"2715"},{"size":"0.1","price":"2711"},{"size":"1","price":"2710"},{"size":"2","price":"2708"},{"size":"51.613","price":"2705"},{"size":"0.01","price":"2701.5"},{"size":"0.2","price":"2701.1"},{"size":"1","price":"2701"},{"size":"0.05","price":"2700.1"},{"size":"106","price":"2700"},{"size":"1","price":"2688"},{"size":"2.5","price":"2686"},{"size":"0.089","price":"2681"},{"size":"2","price":"2680"},{"size":"0.04","price":"2676"},{"size":"0.45","price":"2675"},{"size":"35","price":"2667"},{"size":"4.2","price":"2666"},{"size":"1","price":"2661"},{"size":"1.05","price":"2660"},{"size":"0.2","price":"2655"},{"size":"2","price":"2653"},{"size":"0.6","price":"2651"},{"size":"2.8","price":"2650"},{"size":"15","price":"2649"},{"size":"5","price":"2643"},{"size":"1","price":"2640"},{"size":"0.3","price":"2631"},{"size":"10","price":"2625"},{"size":"0.35","price":"2622"},{"size":"1","price":"2620"},{"size":"0.14","price":"2614"},{"size":"2","price":"2613"},{"size":"5.3","price":"2610"},{"size":"2","price":"2609.8"},{"size":"3","price":"2607"},{"size":"30","price":"2606"},{"size":"2.5","price":"2605"},{"size":"0.2","price":"2601.1"},{"size":"19.6","price":"2600.1"},{"size":"84.47","price":"2600"},{"size":"10.1","price":"2580"},{"size":"2","price":"2577"},{"size":"0.05","price":"2560"},{"size":"3","price":"2556"},{"size":"1","price":"2555"},{"size":"2.5","price":"2552.5"},{"size":"21","price":"2550"},{"size":"0.22","price":"2548"},{"size":"500","price":"2542.8"},{"size":"5","price":"2542"},{"size":"0.4","price":"2522"},{"size":"40","price":"2510"},{"size":"15","price":"2505"},{"size":"5.5","price":"2501"},{"size":"0.5","price":"2500.2"},{"size":"0.1","price":"2500.1"},{"size":"223.26","price":"2500"},{"size":"0.33","price":"2499"},{"size":"10","price":"2476"},{"size":"0.01","price":"2468"},{"size":"207","price":"2455"},{"size":"2.5","price":"2452"},{"size":"1.3","price":"2450"},{"size":"5","price":"2448.3"},{"size":"0.4","price":"2444"},{"size":"1","price":"2438"},{"size":"1","price":"2422.3"},{"size":"0.5","price":"2420"},{"size":"2.5","price":"2417.5"},{"size":"5","price":"2412"},{"size":"0.78","price":"2410"},{"size":"2","price":"2404"},{"size":"0.1","price":"2400.1"},{"size":"367.4","price":"2400"},{"size":"4","price":"2377"},{"size":"20","price":"2360"},{"size":"3","price":"2350"},{"size":"4","price":"2334"},{"size":"6.646","price":"2330"},{"size":"250","price":"2323"},{"size":"0.1","price":"2300.1"},{"size":"40.1","price":"2300"},{"size":"0.4","price":"2250"},{"size":"5","price":"2237"},{"size":"15","price":"2225"},{"size":"5.1","price":"2222"},{"size":"0.5","price":"2220"},{"size":"0.4","price":"2205"},{"size":"0.1","price":"2200.1"},{"size":"3.6","price":"2200"},{"size":"0.5","price":"2150"},{"size":"0.1","price":"2100.1"},{"size":"0.5","price":"2100"},{"size":"10","price":"2072"},{"size":"0.4","price":"2051"},{"size":"0.5","price":"2050"},{"size":"15","price":"2032"},{"size":"20","price":"2022"},{"size":"250","price":"2021"},{"size":"0.1","price":"2006"},{"size":"0.6","price":"2000.1"},{"size":"2.182","price":"2000"},{"size":"20","price":"1995"},{"size":"0.5","price":"1993"},{"size":"0.1","price":"1950"},{"size":"15","price":"1942"},{"size":"15","price":"1922"},{"size":"50","price":"1912"},{"size":"0.2","price":"1900.1"},{"size":"3","price":"1900"},{"size":"0.1","price":"1890"},{"size":"15","price":"1842"},{"size":"2","price":"1827"},{"size":"85","price":"1811"},{"size":"0.2","price":"1800.1"},{"size":"0.56","price":"1800"},{"size":"1","price":"1769"},{"size":"25","price":"1752"},{"size":"0.1","price":"1750"},{"size":"10","price":"1733"},{"size":"150","price":"1711"},{"size":"500","price":"1700.4"},{"size":"0.05","price":"1700"},{"size":"0.4","price":"1660"},{"size":"0.1","price":"1650"},{"size":"20","price":"1632"},{"size":"0.25","price":"1551"},{"size":"0.25","price":"1505.1"},{"size":"0.5","price":"1500.1"},{"size":"30","price":"1500"},{"size":"25","price":"1453"},{"size":"0.2","price":"1450"},{"size":"1","price":"1420"},{"size":"40","price":"1411"},{"size":"40","price":"1381"},{"size":"1.5","price":"1277"},{"size":"5","price":"1155"},{"size":"20","price":"1056"},{"size":"1","price":"1052.1"},{"size":"0.1","price":"1000"},{"size":"2.5","price":"911"},{"size":"0.65","price":"602.1"},{"size":"4","price":"600"},{"size":"2","price":"506.2"},{"size":"1","price":"402.3"},{"size":"8","price":"400"},{"size":"0.3","price":"323"},{"size":"2","price":"305.1"},{"size":"2.6","price":"300"},{"size":"16","price":"200"},{"size":"4","price":"106.6"},{"size":"0.5","price":"102.1"},{"size":"6","price":"32.1"},{"size":"10","price":"10"},{"size":"5","price":"2.2"},{"size":"0.1","price":"1"}]}}
	collectStart := 0
	collectEnd := 94
	depth.Bids = common.Bids{}
	depth.Asks = common.Asks{}
	depth.ParseTime = time.Now()
	depth.Offset = 0
	msgLen := len(msg)
	currentKey := common.JsonKeyUnknown
	for collectEnd < msgLen {
		if collectStart != 0 {
			if msg[collectEnd] == '"' {
				depth.Market = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				collectEnd += 31
				collectStart = collectEnd
				currentKey = common.JsonKeyAskSize
				break
			}
		} else if msg[collectEnd] == '"' &&
			msg[collectEnd-1] == 'd' &&
			msg[collectEnd-2] == 'i' &&
			msg[collectEnd-3] == '"' {
			collectEnd += 3
			collectStart = collectEnd
		}
		collectEnd++
	}
	if currentKey != common.JsonKeyAskSize {
		return fmt.Errorf("bad msg, market not found, %s", msg)
	}
	bid := [2]float64{}
	ask := [2]float64{}
	for collectEnd < msgLen-2 {
		switch currentKey {
		case common.JsonKeyAskSize:
			if msg[collectEnd] == '"' {
				ask[1], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				collectEnd += 11
				collectStart = collectEnd
				currentKey = common.JsonKeyAskPrice
			}
			break
		case common.JsonKeyAskPrice:
			if msg[collectEnd] == '"' {
				ask[0], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				depth.Asks = depth.Asks.Update(ask)
				if msg[collectEnd+2] == ',' {
					collectEnd += 12
					collectStart = collectEnd
					currentKey = common.JsonKeyAskSize
				} else if msg[collectEnd+2] == ']' {
					collectEnd += 21
					collectStart = collectEnd
					currentKey = common.JsonKeyBidSize
				} else {
					return fmt.Errorf("ask price check, bad msg %s", msg[collectEnd:])
				}
			}
			break
		case common.JsonKeyBidSize:
			if msg[collectEnd] == '"' {
				bid[1], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				collectEnd += 11
				collectStart = collectEnd
				currentKey = common.JsonKeyBidPrice
			}
			break
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == '"' {
				bid[0], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				depth.Bids = depth.Bids.Update(bid)
				if msg[collectEnd+2] == ',' {
					collectEnd += 12
					collectStart = collectEnd
					currentKey = common.JsonKeyBidSize
				} else if msg[collectEnd+2] == ']' {
					depth.WithSnapshotData = true
					return
				} else {
					return fmt.Errorf("bid price check, bad msg %s", msg[collectEnd:])
				}
			}
		}
		collectEnd++
	}
	return nil
}

func UpdateDepth(msg []byte, depth *Depth) (err error) {
	//{"type":"subscribed","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":2,"channel":"v3_orderbook","id":"ETH-USD","contents":{"asks":[{"size":"63.452","price":"3567.5"},{"size":"135.45","price":"3567.6"},{"size":"0.3","price":"3567.9"},{"size":"2.345","price":"3568"},{"size":"1.682","price":"3568.3"},{"size":"65.689","price":"3568.6"},{"size":"0.224","price":"3568.9"},{"size":"2.395","price":"3569.2"},{"size":"1419.301","price":"3569.3"},{"size":"31.372","price":"3569.4"},{"size":"4.3","price":"3569.6"},{"size":"3.782","price":"3569.8"},{"size":"1086.46","price":"3570.1"},{"size":"2.445","price":"3570.2"},{"size":"29.031","price":"3570.7"},{"size":"1.542","price":"3571"},{"size":"3.986","price":"3571.1"},{"size":"13.59","price":"3571.5"},{"size":"6.995","price":"3571.6"},{"size":"1.602","price":"3572"},{"size":"14.9","price":"3572.3"},{"size":"2.1","price":"3572.4"},{"size":"1.89","price":"3572.5"},{"size":"3.537","price":"3572.8"},{"size":"20.991","price":"3573"},{"size":"64.642","price":"3573.5"},{"size":"0.392","price":"3573.7"},{"size":"5160.22","price":"3573.8"},{"size":"0.01","price":"3574.2"},{"size":"2.442","price":"3574.4"},{"size":"32.61","price":"3574.6"},{"size":"4.5","price":"3574.8"},{"size":"82.63","price":"3575"},{"size":"2.492","price":"3575.8"},{"size":"80.541","price":"3576"},{"size":"0.2","price":"3576.5"},{"size":"0.3","price":"3576.8"},{"size":"0.07","price":"3577.1"},{"size":"2.097","price":"3577.4"},{"size":"2.387","price":"3577.7"},{"size":"2.592","price":"3578"},{"size":"0.01","price":"3578.2"},{"size":"0.2","price":"3578.4"},{"size":"0.2","price":"3578.7"},{"size":"0.359","price":"3579.5"},{"size":"2.2","price":"3579.8"},{"size":"8","price":"3580"},{"size":"0.2","price":"3580.1"},{"size":"2.305","price":"3580.4"},{"size":"56.07","price":"3580.5"},{"size":"0.189","price":"3581.1"},{"size":"2.64","price":"3581.8"},{"size":"8.302","price":"3584"},{"size":"0.011","price":"3584.1"},{"size":"2.651","price":"3584.2"},{"size":"14.246","price":"3584.9"},{"size":"0.71","price":"3585.2"},{"size":"2.511","price":"3585.3"},{"size":"2.301","price":"3585.9"},{"size":"2.602","price":"3586.1"},{"size":"2.788","price":"3587.6"},{"size":"8","price":"3588.4"},{"size":"0.01","price":"3590.1"},{"size":"2.785","price":"3590.8"},{"size":"2.507","price":"3591"},{"size":"2.507","price":"3591.4"},{"size":"12","price":"3594"},{"size":"5","price":"3594.4"},{"size":"2.503","price":"3596.7"},{"size":"12","price":"3597"},{"size":"1","price":"3597.3"},{"size":"2","price":"3598.7"},{"size":"2","price":"3598.9"},{"size":"5","price":"3599"},{"size":"2","price":"3599.1"},{"size":"2.5","price":"3599.9"},{"size":"2.3","price":"3600"},{"size":"1","price":"3600.5"},{"size":"1","price":"3602.1"},{"size":"0.01","price":"3602.8"},{"size":"2","price":"3604"},{"size":"0.01","price":"3607"},{"size":"1.6","price":"3608.6"},{"size":"20","price":"3611"},{"size":"3","price":"3613.4"},{"size":"1","price":"3613.5"},{"size":"1","price":"3614.9"},{"size":"1","price":"3617.2"},{"size":"5.3","price":"3620"},{"size":"31","price":"3621"},{"size":"443.15","price":"3625"},{"size":"20","price":"3628"},{"size":"10","price":"3630"},{"size":"1","price":"3631"},{"size":"0.01","price":"3632.5"},{"size":"10","price":"3633"},{"size":"1.88","price":"3636"},{"size":"31.6","price":"3640"},{"size":"2.5","price":"3645"},{"size":"2","price":"3648"},{"size":"8.86","price":"3649"},{"size":"199.2","price":"3650"},{"size":"1.499","price":"3653"},{"size":"25","price":"3655"},{"size":"22","price":"3658"},{"size":"131","price":"3660"},{"size":"0.05","price":"3666"},{"size":"9","price":"3668.3"},{"size":"25","price":"3670"},{"size":"14","price":"3672"},{"size":"46.3","price":"3675"},{"size":"20","price":"3677"},{"size":"1","price":"3679"},{"size":"26","price":"3680"},{"size":"0.01","price":"3684.3"},{"size":"45","price":"3685"},{"size":"4","price":"3688"},{"size":"2","price":"3689"},{"size":"30","price":"3690"},{"size":"0.1","price":"3693"},{"size":"5","price":"3695"},{"size":"1","price":"3699"},{"size":"100.6","price":"3700"},{"size":"5","price":"3705"},{"size":"10","price":"3710"},{"size":"5","price":"3715"},{"size":"5","price":"3725"},{"size":"2","price":"3728.7"},{"size":"4","price":"3730"},{"size":"0.2","price":"3737"},{"size":"0.15","price":"3737.7"},{"size":"0.01","price":"3745"},{"size":"45.403","price":"3750"},{"size":"3","price":"3753"},{"size":"12.65","price":"3757.9"},{"size":"0.171","price":"3766"},{"size":"10.48","price":"3767.2"},{"size":"1","price":"3770"},{"size":"1","price":"3783"},{"size":"1","price":"3785"},{"size":"1","price":"3788"},{"size":"12.65","price":"3795.1"},{"size":"10","price":"3799"},{"size":"50.6","price":"3800"},{"size":"10.48","price":"3804.5"},{"size":"30","price":"3833"},{"size":"30","price":"3855"},{"size":"1","price":"3870"},{"size":"63","price":"3888"},{"size":"36.01","price":"3900"},{"size":"5","price":"3907.1"},{"size":"0.3","price":"3910"},{"size":"7.312","price":"3930"},{"size":"0.1","price":"3934"},{"size":"0.25","price":"3940"},{"size":"13","price":"3945"},{"size":"20.5","price":"3950"},{"size":"1","price":"3970"},{"size":"0.52","price":"3980"},{"size":"7.5","price":"3987"},{"size":"50.586","price":"3999"},{"size":"173.653","price":"4000"},{"size":"34.89","price":"4019.9"},{"size":"9.757","price":"4100"},{"size":"0.5","price":"4128.2"},{"size":"0.02","price":"4150"},{"size":"1.5","price":"4193"},{"size":"2","price":"4200"},{"size":"1","price":"4210"},{"size":"3.128","price":"4236"},{"size":"0.05","price":"4298"},{"size":"40.011","price":"4300"},{"size":"10","price":"4325"},{"size":"16","price":"4400"},{"size":"174.22","price":"4500"},{"size":"80","price":"4560"},{"size":"10","price":"4600"},{"size":"0.08","price":"4666"},{"size":"25","price":"4711"},{"size":"11.49","price":"4791"},{"size":"150","price":"4796.3"},{"size":"75","price":"4820"},{"size":"200","price":"4944"},{"size":"0.1","price":"4999"},{"size":"500","price":"4999.3"},{"size":"65","price":"5150"},{"size":"0.01","price":"5555"},{"size":"3","price":"6000"},{"size":"500","price":"6800"},{"size":"3","price":"6984"},{"size":"0.55","price":"6990"},{"size":"80","price":"8899.8"}],"bids":[{"size":"15","price":"3567.4"},{"size":"0.5","price":"3567.3"},{"size":"15","price":"3567"},{"size":"1.683","price":"3566.8"},{"size":"2.72","price":"3566.7"},{"size":"1.66","price":"3566.5"},{"size":"153.044","price":"3566.4"},{"size":"6.595","price":"3566.2"},{"size":"33.33","price":"3566.1"},{"size":"31.794","price":"3565.9"},{"size":"2.346","price":"3565.7"},{"size":"1100.47","price":"3565.5"},{"size":"2.397","price":"3565.3"},{"size":"29.33","price":"3565.1"},{"size":"57.631","price":"3564.8"},{"size":"4.655","price":"3564.7"},{"size":"28.58","price":"3564.1"},{"size":"15.052","price":"3564"},{"size":"21.044","price":"3563.9"},{"size":"4.252","price":"3563.8"},{"size":"50.73","price":"3563.5"},{"size":"49.114","price":"3563.1"},{"size":"1.802","price":"3563"},{"size":"2.246","price":"3562.2"},{"size":"5139.221","price":"3561.9"},{"size":"0.2","price":"3561.8"},{"size":"0.22","price":"3561.6"},{"size":"0.2","price":"3561.5"},{"size":"2.855","price":"3561.2"},{"size":"2.528","price":"3561.1"},{"size":"84.105","price":"3560.8"},{"size":"1","price":"3560.1"},{"size":"1.5","price":"3560"},{"size":"2.605","price":"3559.9"},{"size":"0.2","price":"3559.6"},{"size":"2.389","price":"3559.2"},{"size":"0.4","price":"3559"},{"size":"0.011","price":"3558.4"},{"size":"84.105","price":"3558"},{"size":"2.709","price":"3557.8"},{"size":"1","price":"3557.5"},{"size":"0.14","price":"3556.9"},{"size":"0.227","price":"3556.3"},{"size":"2.762","price":"3555.7"},{"size":"0.14","price":"3555.3"},{"size":"0.4","price":"3553"},{"size":"87.574","price":"3552.4"},{"size":"2.816","price":"3552"},{"size":"2.817","price":"3550.9"},{"size":"86","price":"3550"},{"size":"0.8","price":"3549.2"},{"size":"14.002","price":"3548.7"},{"size":"2.818","price":"3548.6"},{"size":"2.819","price":"3548.3"},{"size":"12.797","price":"3548.2"},{"size":"11","price":"3546"},{"size":"0.2","price":"3545"},{"size":"1","price":"3543.3"},{"size":"15","price":"3535"},{"size":"1.6","price":"3533.5"},{"size":"30.524","price":"3530"},{"size":"1","price":"3529.2"},{"size":"5.1","price":"3528"},{"size":"2","price":"3527"},{"size":"3","price":"3526.7"},{"size":"15","price":"3524"},{"size":"1","price":"3522"},{"size":"20","price":"3520"},{"size":"1","price":"3518.1"},{"size":"10.1","price":"3518"},{"size":"2","price":"3517"},{"size":"1","price":"3515.7"},{"size":"1","price":"3515.1"},{"size":"0.3","price":"3515"},{"size":"2","price":"3512.3"},{"size":"15","price":"3512"},{"size":"33.5","price":"3510"},{"size":"17","price":"3509"},{"size":"22","price":"3508"},{"size":"1","price":"3507"},{"size":"15","price":"3505"},{"size":"120","price":"3502"},{"size":"1","price":"3501.1"},{"size":"2.5","price":"3500.1"},{"size":"42.35","price":"3500"},{"size":"0.05","price":"3495"},{"size":"1","price":"3493.9"},{"size":"30","price":"3492"},{"size":"15","price":"3489"},{"size":"17","price":"3488"},{"size":"19","price":"3487"},{"size":"1","price":"3482"},{"size":"12","price":"3481"},{"size":"51.51","price":"3480"},{"size":"0.1","price":"3475"},{"size":"1.5","price":"3470"},{"size":"17","price":"3469"},{"size":"15","price":"3468"},{"size":"12","price":"3466"},{"size":"10","price":"3461.2"},{"size":"1","price":"3460"},{"size":"12","price":"3456"},{"size":"32","price":"3455"},{"size":"2.5","price":"3450.1"},{"size":"62","price":"3450"},{"size":"10","price":"3449"},{"size":"5","price":"3448"},{"size":"1","price":"3445"},{"size":"1","price":"3440"},{"size":"2","price":"3436"},{"size":"15","price":"3433"},{"size":"1.2","price":"3430"},{"size":"2","price":"3425"},{"size":"4","price":"3424.2"},{"size":"32","price":"3423"},{"size":"25","price":"3422"},{"size":"25","price":"3421.8"},{"size":"5","price":"3420.1"},{"size":"0.3","price":"3420"},{"size":"18","price":"3417"},{"size":"1","price":"3415"},{"size":"6.5","price":"3414"},{"size":"0.4","price":"3412"},{"size":"32","price":"3411"},{"size":"1","price":"3410"},{"size":"5","price":"3408"},{"size":"1","price":"3404"},{"size":"2","price":"3403"},{"size":"5","price":"3402.1"},{"size":"2.5","price":"3400.1"},{"size":"55.5","price":"3400"},{"size":"2","price":"3395"},{"size":"17","price":"3389"},{"size":"15","price":"3387"},{"size":"10","price":"3385"},{"size":"45","price":"3382"},{"size":"10","price":"3381.1"},{"size":"20","price":"3376"},{"size":"1","price":"3372"},{"size":"1","price":"3370"},{"size":"22","price":"3369"},{"size":"0.3","price":"3367"},{"size":"1.5","price":"3363"},{"size":"33.78","price":"3359.3"},{"size":"0.7","price":"3358"},{"size":"1","price":"3356"},{"size":"10","price":"3355"},{"size":"10","price":"3351.1"},{"size":"5","price":"3350.1"},{"size":"6.2","price":"3350"},{"size":"45.25","price":"3346"},{"size":"15","price":"3335.5"},{"size":"3.991","price":"3333"},{"size":"33.78","price":"3321.8"},{"size":"10","price":"3321.1"},{"size":"0.02","price":"3320"},{"size":"0.5","price":"3318"},{"size":"2.2","price":"3315"},{"size":"0.02","price":"3311"},{"size":"5","price":"3310"},{"size":"1.5","price":"3308.2"},{"size":"6.5","price":"3308"},{"size":"5","price":"3304.6"},{"size":"0.01","price":"3301.5"},{"size":"2.5","price":"3300.1"},{"size":"66.5","price":"3300"},{"size":"15","price":"3291.1"},{"size":"420.1","price":"3280"},{"size":"0.1","price":"3278.7"},{"size":"50","price":"3276"},{"size":"0.01","price":"3275"},{"size":"1","price":"3270"},{"size":"0.5","price":"3265"},{"size":"20","price":"3255"},{"size":"2","price":"3254"},{"size":"0.5","price":"3253"},{"size":"1","price":"3252"},{"size":"0.01","price":"3251.5"},{"size":"31.15","price":"3250"},{"size":"0.5","price":"3242.6"},{"size":"12.25","price":"3240"},{"size":"0.5","price":"3235"},{"size":"2","price":"3230"},{"size":"5","price":"3225"},{"size":"0.65","price":"3222"},{"size":"3","price":"3220"},{"size":"0.025","price":"3217"},{"size":"0.05","price":"3216.1"},{"size":"3.5","price":"3210"},{"size":"10","price":"3209"},{"size":"5","price":"3208"},{"size":"30","price":"3207"},{"size":"8","price":"3206"},{"size":"0.01","price":"3201.5"},{"size":"0.05","price":"3200.1"},{"size":"47.2","price":"3200"},{"size":"2","price":"3198"},{"size":"3","price":"3195.5"},{"size":"30","price":"3189"},{"size":"8","price":"3185"},{"size":"5","price":"3183"},{"size":"1.1","price":"3180"},{"size":"1","price":"3176"},{"size":"3","price":"3175.9"},{"size":"1","price":"3170"},{"size":"1","price":"3169"},{"size":"3","price":"3168"},{"size":"51","price":"3166"},{"size":"0.1","price":"3165.8"},{"size":"0.5","price":"3164"},{"size":"4","price":"3160"},{"size":"12","price":"3155"},{"size":"1.15","price":"3153"},{"size":"0.01","price":"3151.5"},{"size":"13","price":"3151"},{"size":"0.05","price":"3150.1"},{"size":"2.8","price":"3150"},{"size":"3","price":"3146"},{"size":"2","price":"3142"},{"size":"1.75","price":"3132"},{"size":"20","price":"3130.1"},{"size":"50","price":"3126"},{"size":"55","price":"3125"},{"size":"50","price":"3124"},{"size":"50","price":"3123"},{"size":"50.1","price":"3122"},{"size":"1","price":"3120"},{"size":"0.4","price":"3111"},{"size":"1","price":"3110"},{"size":"5.5","price":"3108"},{"size":"0.01","price":"3101.5"},{"size":"0.05","price":"3100.1"},{"size":"308.1","price":"3100"},{"size":"4","price":"3099"},{"size":"0.2","price":"3098"},{"size":"60","price":"3077"},{"size":"0.25","price":"3075"},{"size":"1.1","price":"3070"},{"size":"20","price":"3066"},{"size":"0.01","price":"3051.5"},{"size":"0.05","price":"3050.1"},{"size":"0.4","price":"3050"},{"size":"0.1","price":"3033"},{"size":"2","price":"3030"},{"size":"2","price":"3025"},{"size":"10.15","price":"3022"},{"size":"5","price":"3021"},{"size":"0.5","price":"3018"},{"size":"6","price":"3008"},{"size":"0.4","price":"3007"},{"size":"0.11","price":"3003"},{"size":"0.01","price":"3001.5"},{"size":"115","price":"3001"},{"size":"0.05","price":"3000.1"},{"size":"9.876","price":"3000"},{"size":"0.35","price":"2999"},{"size":"1","price":"2996"},{"size":"0.03","price":"2986"},{"size":"0.5","price":"2984"},{"size":"10","price":"2980"},{"size":"0.05","price":"2978"},{"size":"0.7","price":"2976"},{"size":"10","price":"2958"},{"size":"0.18","price":"2955"},{"size":"0.01","price":"2951.5"},{"size":"76.5","price":"2950"},{"size":"1","price":"2948"},{"size":"2","price":"2945"},{"size":"0.1","price":"2940"},{"size":"2","price":"2938"},{"size":"0.05","price":"2933.1"},{"size":"0.1","price":"2930"},{"size":"8.39","price":"2927"},{"size":"0.2","price":"2922"},{"size":"0.1","price":"2920"},{"size":"0.1","price":"2917.3"},{"size":"0.1","price":"2917.1"},{"size":"0.1","price":"2917"},{"size":"1","price":"2911"},{"size":"0.6","price":"2910"},{"size":"8","price":"2908"},{"size":"0.05","price":"2905"},{"size":"20","price":"2903"},{"size":"0.01","price":"2901.5"},{"size":"0.05","price":"2900.1"},{"size":"71.42","price":"2900"},{"size":"0.81","price":"2892"},{"size":"0.1","price":"2890"},{"size":"0.1","price":"2880"},{"size":"300","price":"2871"},{"size":"0.1","price":"2870"},{"size":"3","price":"2861"},{"size":"0.1","price":"2860"},{"size":"0.01","price":"2851.5"},{"size":"172.35","price":"2850"},{"size":"0.1","price":"2840"},{"size":"0.9","price":"2833"},{"size":"0.1","price":"2830"},{"size":"0.15","price":"2824.8"},{"size":"0.1","price":"2824"},{"size":"0.25","price":"2822"},{"size":"6.1","price":"2820"},{"size":"0.5","price":"2818"},{"size":"0.1","price":"2810"},{"size":"5","price":"2808"},{"size":"0.215","price":"2805.1"},{"size":"0.01","price":"2801.5"},{"size":"0.05","price":"2800.1"},{"size":"101.7","price":"2800"},{"size":"1","price":"2788"},{"size":"8","price":"2786.1"},{"size":"5","price":"2781"},{"size":"1","price":"2777"},{"size":"2","price":"2758.8"},{"size":"1","price":"2758.3"},{"size":"0.593","price":"2755"},{"size":"0.01","price":"2751.5"},{"size":"9.45","price":"2750"},{"size":"10","price":"2745"},{"size":"12","price":"2743"},{"size":"1","price":"2740"},{"size":"60","price":"2739"},{"size":"0.5","price":"2730"},{"size":"0.2","price":"2727"},{"size":"3","price":"2725"},{"size":"0.3","price":"2722"},{"size":"1","price":"2720"},{"size":"1.4","price":"2718"},{"size":"5.1","price":"2715"},{"size":"0.1","price":"2711"},{"size":"1","price":"2710"},{"size":"2","price":"2708"},{"size":"51.613","price":"2705"},{"size":"0.01","price":"2701.5"},{"size":"0.2","price":"2701.1"},{"size":"1","price":"2701"},{"size":"0.05","price":"2700.1"},{"size":"106","price":"2700"},{"size":"1","price":"2688"},{"size":"2.5","price":"2686"},{"size":"0.089","price":"2681"},{"size":"2","price":"2680"},{"size":"0.04","price":"2676"},{"size":"0.45","price":"2675"},{"size":"35","price":"2667"},{"size":"4.2","price":"2666"},{"size":"1","price":"2661"},{"size":"1.05","price":"2660"},{"size":"0.2","price":"2655"},{"size":"2","price":"2653"},{"size":"0.6","price":"2651"},{"size":"2.8","price":"2650"},{"size":"15","price":"2649"},{"size":"5","price":"2643"},{"size":"1","price":"2640"},{"size":"0.3","price":"2631"},{"size":"10","price":"2625"},{"size":"0.35","price":"2622"},{"size":"1","price":"2620"},{"size":"0.14","price":"2614"},{"size":"2","price":"2613"},{"size":"5.3","price":"2610"},{"size":"2","price":"2609.8"},{"size":"3","price":"2607"},{"size":"30","price":"2606"},{"size":"2.5","price":"2605"},{"size":"0.2","price":"2601.1"},{"size":"19.6","price":"2600.1"},{"size":"84.47","price":"2600"},{"size":"10.1","price":"2580"},{"size":"2","price":"2577"},{"size":"0.05","price":"2560"},{"size":"3","price":"2556"},{"size":"1","price":"2555"},{"size":"2.5","price":"2552.5"},{"size":"21","price":"2550"},{"size":"0.22","price":"2548"},{"size":"500","price":"2542.8"},{"size":"5","price":"2542"},{"size":"0.4","price":"2522"},{"size":"40","price":"2510"},{"size":"15","price":"2505"},{"size":"5.5","price":"2501"},{"size":"0.5","price":"2500.2"},{"size":"0.1","price":"2500.1"},{"size":"223.26","price":"2500"},{"size":"0.33","price":"2499"},{"size":"10","price":"2476"},{"size":"0.01","price":"2468"},{"size":"207","price":"2455"},{"size":"2.5","price":"2452"},{"size":"1.3","price":"2450"},{"size":"5","price":"2448.3"},{"size":"0.4","price":"2444"},{"size":"1","price":"2438"},{"size":"1","price":"2422.3"},{"size":"0.5","price":"2420"},{"size":"2.5","price":"2417.5"},{"size":"5","price":"2412"},{"size":"0.78","price":"2410"},{"size":"2","price":"2404"},{"size":"0.1","price":"2400.1"},{"size":"367.4","price":"2400"},{"size":"4","price":"2377"},{"size":"20","price":"2360"},{"size":"3","price":"2350"},{"size":"4","price":"2334"},{"size":"6.646","price":"2330"},{"size":"250","price":"2323"},{"size":"0.1","price":"2300.1"},{"size":"40.1","price":"2300"},{"size":"0.4","price":"2250"},{"size":"5","price":"2237"},{"size":"15","price":"2225"},{"size":"5.1","price":"2222"},{"size":"0.5","price":"2220"},{"size":"0.4","price":"2205"},{"size":"0.1","price":"2200.1"},{"size":"3.6","price":"2200"},{"size":"0.5","price":"2150"},{"size":"0.1","price":"2100.1"},{"size":"0.5","price":"2100"},{"size":"10","price":"2072"},{"size":"0.4","price":"2051"},{"size":"0.5","price":"2050"},{"size":"15","price":"2032"},{"size":"20","price":"2022"},{"size":"250","price":"2021"},{"size":"0.1","price":"2006"},{"size":"0.6","price":"2000.1"},{"size":"2.182","price":"2000"},{"size":"20","price":"1995"},{"size":"0.5","price":"1993"},{"size":"0.1","price":"1950"},{"size":"15","price":"1942"},{"size":"15","price":"1922"},{"size":"50","price":"1912"},{"size":"0.2","price":"1900.1"},{"size":"3","price":"1900"},{"size":"0.1","price":"1890"},{"size":"15","price":"1842"},{"size":"2","price":"1827"},{"size":"85","price":"1811"},{"size":"0.2","price":"1800.1"},{"size":"0.56","price":"1800"},{"size":"1","price":"1769"},{"size":"25","price":"1752"},{"size":"0.1","price":"1750"},{"size":"10","price":"1733"},{"size":"150","price":"1711"},{"size":"500","price":"1700.4"},{"size":"0.05","price":"1700"},{"size":"0.4","price":"1660"},{"size":"0.1","price":"1650"},{"size":"20","price":"1632"},{"size":"0.25","price":"1551"},{"size":"0.25","price":"1505.1"},{"size":"0.5","price":"1500.1"},{"size":"30","price":"1500"},{"size":"25","price":"1453"},{"size":"0.2","price":"1450"},{"size":"1","price":"1420"},{"size":"40","price":"1411"},{"size":"40","price":"1381"},{"size":"1.5","price":"1277"},{"size":"5","price":"1155"},{"size":"20","price":"1056"},{"size":"1","price":"1052.1"},{"size":"0.1","price":"1000"},{"size":"2.5","price":"911"},{"size":"0.65","price":"602.1"},{"size":"4","price":"600"},{"size":"2","price":"506.2"},{"size":"1","price":"402.3"},{"size":"8","price":"400"},{"size":"0.3","price":"323"},{"size":"2","price":"305.1"},{"size":"2.6","price":"300"},{"size":"16","price":"200"},{"size":"4","price":"106.6"},{"size":"0.5","price":"102.1"},{"size":"6","price":"32.1"},{"size":"10","price":"10"},{"size":"5","price":"2.2"},{"size":"0.1","price":"1"}]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":3,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724605","bids":[],"asks":[["3568.9","12.987"]]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":4,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724615","bids":[["3567.4","17.72"],["3566.7","0"]],"asks":[]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":5,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724613","bids":[],"asks":[["3569","1.787"]]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":6,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933723364","bids":[],"asks":[["3569.8","7.504"]]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":7,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724626","bids":[["3566.2","6.819"]],"asks":[]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":8,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724628","bids":[],"asks":[["3598.9","0"]]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":9,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724622","bids":[],"asks":[["3567.5","60.742"],["3568.3","4.402"]]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":10,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724634","bids":[],"asks":[["3568.5","0"]]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":11,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724629","bids":[["3566.5","0"]],"asks":[]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":12,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933724637","bids":[["3566.3","1.52"]],"asks":[]}}
	//{"type":"channel_data","connection_id":"83d08710-eb85-4718-9ce1-d09ac31ebf77","message_id":13,"id":"ETH-USD","channel":"v3_orderbook","contents":{"offset":"1933723405","bids":[],"asks":[["3569.8","9.395"]]}}
	msgLen := len(msg)
	if msgLen < 128 {
		return fmt.Errorf("len check, bad msg %s", msg)
	}
	if msg[9] == 's' && msg[18] == 'd' {
		return ParseDepth(msg, depth)
	} else if msg[9] != 'c' || msg[20] != 'a' {
		return fmt.Errorf("bad msg %s", msg)
	}

	collectStart := 0
	collectEnd := 94
	depth.ParseTime = time.Now()
	currentKey := common.JsonKeyUnknown
	for collectEnd < msgLen {
		if collectStart != 0 {
			if msg[collectEnd] == '"' {
				depth.Market = common.UnsafeBytesToString(msg[collectStart:collectEnd])
				collectEnd += 45
				collectStart = collectEnd
				break
			}
		} else if msg[collectEnd] == '"' &&
			msg[collectEnd-1] == 'd' &&
			msg[collectEnd-2] == 'i' &&
			msg[collectEnd-3] == '"' {
			collectEnd += 3
			collectStart = collectEnd
		}
		collectEnd++
	}
	bid := [2]float64{}
	ask := [2]float64{}
	for collectEnd < msgLen {
		switch currentKey {
		case common.JsonKeyBidPrice:
			if msg[collectEnd] == '"' {
				bid[0], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				collectEnd += 3
				collectStart = collectEnd
				currentKey = common.JsonKeyBidSize
			}
			break
		case common.JsonKeyBidSize:
			if msg[collectEnd] == '"' {
				bid[1], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return err
				}
				depth.Bids = depth.Bids.Update(bid)
				collectEnd += 2
				if collectEnd < msgLen {
					if msg[collectEnd] == ',' {
						//还有bid
						currentKey = common.JsonKeyBidPrice
						collectEnd += 3
						collectStart = collectEnd
					} else if msg[collectEnd] == ']' {
						//已经结束
						collectEnd += 10
						if collectEnd < msgLen {
							if msg[collectEnd] == '[' {
								//ask不为空
								currentKey = common.JsonKeyAskPrice
								collectEnd += 2
								collectStart = collectEnd
							} else if msg[collectEnd] == ']' {
								//ask为空, 解析结束
								return
							} else {
								return fmt.Errorf("bad ask %s", msg[collectStart:])
							}
						} else {
							return fmt.Errorf("msg too short, %s", msg)
						}
					} else {
						return fmt.Errorf("bad bid %s", msg[collectStart:])
					}
				} else {
					return fmt.Errorf("msg too short, %s", msg)
				}
			}
			break
		case common.JsonKeyAskPrice:
			if msg[collectEnd] == '"' {
				ask[0], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				collectEnd += 3
				collectStart = collectEnd
				currentKey = common.JsonKeyAskSize
			}
			break
		case common.JsonKeyAskSize:
			if msg[collectEnd] == '"' {
				ask[1], err = common.ParseDecimal(msg[collectStart:collectEnd])
				if err != nil {
					return
				}
				depth.Asks = depth.Asks.Update(ask)
				collectEnd += 2
				if collectEnd < msgLen {
					if msg[collectEnd] == ',' {
						//还有ask
						currentKey = common.JsonKeyAskPrice
						collectEnd += 3
						collectStart = collectEnd
					} else if msg[collectEnd] == ']' {
						//ask结束
						return
					} else {
						return fmt.Errorf("bad ask end %s", msg[collectStart:])
					}
				} else {
					return fmt.Errorf("msg too short, %s", msg)
				}
			}
			break
		case common.JsonKeyID:
			if msg[collectEnd] == '"' {
				newOffset, err := common.ParseInt(msg[collectStart:collectEnd])
				if err != nil {
					return err
				}else if newOffset < depth.Offset {
					return fmt.Errorf("old msg offset %d < %d", newOffset, depth.Offset)
				}else{
					depth.Offset = newOffset
					currentKey = common.JsonKeyUnknown
				}
			}
			break
		case common.JsonKeyUnknown:
			if msg[collectEnd] == 't' && msg[collectEnd-6] == 'o' && msg[collectEnd-1] == 'e' {
				currentKey = common.JsonKeyID
				collectEnd += 4
				collectStart = collectEnd
			}else if msg[collectEnd] == 's' && msg[collectEnd-1] == 'd' && msg[collectEnd-3] == 'b' {
				if msg[collectEnd+4] == '[' {
					collectEnd += 6
					collectStart = collectEnd
					currentKey = common.JsonKeyBidPrice
				} else if msg[collectEnd+4] == ']' {
					//没有bids
					collectEnd += 14
					if msg[collectEnd] == '[' {
						//ask不为空
						currentKey = common.JsonKeyAskPrice
						collectEnd += 2
						collectStart = collectEnd
					} else if msg[collectEnd] == ']' {
						//ask为空, 解析结束
						return
					} else {
						return fmt.Errorf("bad ask %s", msg[collectStart:])
					}

				} else {
					return fmt.Errorf("bad bids %s", msg)
				}
			}
			break
		}
		collectEnd += 1
	}
	return nil
}
