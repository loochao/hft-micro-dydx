package starkex

//def get_y_coordinate(stark_key_x_coordinate: int) -> int:
//    """
//    Given the x coordinate of a stark_key, returns a possible y coordinate such that together the
//    point (x,y) is on the curve.
//    Note that the real y coordinate is either y or -y.
//    If x is invalid stark_key it throws an error.
//    """
//
//    x = stark_key_x_coordinate
//    y_squared = (x * x * x + ALPHA * x + BETA) % FIELD_PRIME
//    if not is_quad_residue(y_squared, FIELD_PRIME):
//        raise InvalidPublicKeyError()
//    return sqrt_mod(y_squared, FIELD_PRIME)

//func GetYCoordinate(starkKeyXCoordinate Decimal256) int {
//	//    Given the x coordinate of a stark_key, returns a possible y coordinate such that together the
//	//    point (x,y) is on the curve.
//	//    Note that the real y coordinate is either y or -y.
//	//    If x is invalid stark_key it throws an error.
//	x := starkKeyXCoordinate
//	ySquared := (x*x*x + PedersenParams.Alpha*x + PedersenParams.Beta) % PedersenParams.FieldPrime
//}
