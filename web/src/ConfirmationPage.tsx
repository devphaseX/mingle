import { useNavigate, useParams } from "react-router-dom";

const API_URL = import.meta.env["VITE_BACKEND_URL"] || "http://localhost:8080";
export const ConfirmationPage = () => {
  const { token = "" } = useParams();
  const redirect = useNavigate();
  const handleConfirm = async () => {
    const response = await fetch(`${API_URL}/users/activate/${token}`, {
      method: "PUT",
    });

    if (response.ok) {
      //
      return redirect("/");
    }

    alert("failed to confirm token");
  };
  return (
    <div>
      <h1>Confirmation</h1>
      <button onClick={handleConfirm}>Click to confirm</button>
    </div>
  );
};
