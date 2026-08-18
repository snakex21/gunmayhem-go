stop();
this.onRollOver = function()
{
   gotoAndPlay(2);
   _parent.allplay();
};
this.onRollOut = function()
{
   gotoAndPlay(9);
};
this.onRelease = function()
{
   _parent.alldisable();
   chosen._alpha = 100;
   switch(_name)
   {
      case "mapbtn0":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 0);
         _root.mapnumber = 0;
         break;
      case "mapbtn1":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 1);
         _root.mapnumber = _parent.mapdisplay._totalframes - 1;
         break;
      case "mapbtn2":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 2);
         _root.mapnumber = _parent.mapdisplay._totalframes - 2;
         break;
      case "mapbtn3":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 3);
         _root.mapnumber = _parent.mapdisplay._totalframes - 3;
         break;
      case "mapbtn4":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 4);
         _root.mapnumber = _parent.mapdisplay._totalframes - 4;
         break;
      case "mapbtn5":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 5);
         _root.mapnumber = _parent.mapdisplay._totalframes - 5;
         break;
      case "mapbtn6":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 6);
         _root.mapnumber = _parent.mapdisplay._totalframes - 6;
         break;
      case "mapbtn7":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 7);
         _root.mapnumber = _parent.mapdisplay._totalframes - 7;
         break;
      case "mapbtn8":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 8);
         _root.mapnumber = _parent.mapdisplay._totalframes - 8;
         break;
      case "mapbtn9":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 9);
         _root.mapnumber = _parent.mapdisplay._totalframes - 9;
         break;
      case "mapbtn10":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 10);
         _root.mapnumber = _parent.mapdisplay._totalframes - 10;
         break;
      case "mapbtn11":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 11);
         _root.mapnumber = _parent.mapdisplay._totalframes - 11;
         break;
      case "mapbtn12":
         _parent.mapdisplay.gotoAndStop(_parent.mapdisplay._totalframes - 12);
         _root.mapnumber = _parent.mapdisplay._totalframes - 12;
      default:
         return;
   }
};
