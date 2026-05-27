import axios from "axios";

/*
Fetch live graph state
from Go observability engine.
*/
export async function fetchGraph() {
  const response = await axios.get("http://localhost:8090/graph");

  return response.data;
}
